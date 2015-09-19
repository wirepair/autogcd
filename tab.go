package autogcd

import (
	"encoding/base64"
	"encoding/json"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
	"log"
	"sync"
	"time"
)

// When we are unable to find an element/nodeId
type ElementNotFoundErr struct {
	Message string
}

func (e *ElementNotFoundErr) Error() string {
	return "Unable to find element: " + e.Message
}

// When we are unable to access a tab
type InvalidTabErr struct {
	Message string
}

func (e *InvalidTabErr) Error() string {
	return "Unable to access tab: " + e.Message
}

// When Tab.Navigate has timed out
type NavigationTimeoutErr struct {
	Message string
}

func (e *NavigationTimeoutErr) Error() string {
	return "Timed out attempting to navigate to: " + e.Message
}

type GcdResponseFunc func(target *gcd.ChromeTarget, payload []byte)

// A function to handle javascript dialog prompts as they occur, pass to SetJavaScriptPromptHandler
// Internally this should call tab.Page.HandleJavaScriptDialog(accept bool, promptText string)
type PromptHandlerFunc func(tab *Tab, message, promptType string)

// A function for handling console messages
type ConsoleMessageFunc func(tab *Tab, message *gcdapi.ConsoleConsoleMessage)

// A function for handling network requests
type NetworkRequestHandlerFunc func(tab *Tab, request *NetworkRequest)

// A function for handling network responses
type NetworkResponseHandlerFunc func(tab *Tab, response *NetworkResponse)

// A function for ListenStorageEvents returns the eventType of cleared, updated, removed or added.
type StorageFunc func(tab *Tab, eventType string, eventDetails *StorageEvent)

type Tab struct {
	*gcd.ChromeTarget
	eleMutex          *sync.RWMutex
	Elements          map[int]*Element
	nodeChange        chan *NodeChangeEvent
	navigationTimeout time.Duration
}

func NewTab(target *gcd.ChromeTarget) *Tab {
	t := &Tab{ChromeTarget: target}
	t.eleMutex = &sync.RWMutex{}
	t.Elements = make(map[int]*Element)
	t.nodeChange = make(chan *NodeChangeEvent)
	t.navigationTimeout = 30 // default 30 seconds for timeout
	t.Page.Enable()
	t.DOM.Enable()
	t.Console.Enable()
	t.Debugger.Enable()
	t.subscribeNodeChanges()
	go t.listenNodeChanges()
	return t
}

func (t *Tab) SetNavigationTimeout(timeout time.Duration) {
	t.navigationTimeout = timeout
}

// Navigates to a URL and does not return until the Page.loadEventFired event.
// Updates the list of elements
// Returns the frameId of the Tab that this navigation occured on.
func (t *Tab) Navigate(url string) (string, error) {
	//var doc *gcdapi.DOMNode
	resp := make(chan int, 1)
	t.Subscribe("Page.loadEventFired", t.defaultLoadFired(resp))
	frameId, err := t.Page.Navigate(url)
	timeoutTimer := time.NewTimer(t.navigationTimeout * time.Second)
	if err != nil {
		return "", err
	}
	select {
	case <-resp:
		timeoutTimer.Stop()
	case <-timeoutTimer.C:
		return "", &NavigationTimeoutErr{Message: url}
	}
	// update the list of elements
	_, err = t.GetDocument()
	if err != nil {
		return "", err
	}
	//_, err = t.DOM.RequestChildNodes(doc.NodeId, -1)
	return frameId, nil
}

// Gets the document and updates our list of Elements in the event
// they've changed, methods should always internally call this so we
// can be sure our Element list is up to date
func (t *Tab) GetDocument() (*gcdapi.DOMNode, error) {
	doc, err := t.DOM.GetDocument()
	if err != nil {
		return nil, err
	}
	t.addDocumentNodes(doc)
	return doc, nil
}

// Returns the top window documents source, as visible
func (t *Tab) GetPageSource() (string, error) {
	node, err := t.GetDocument()
	if err != nil {
		return "", err
	}
	return t.DOM.GetOuterHTML(node.NodeId)
}

// Returns the source of a script by its scriptId.
func (t *Tab) GetScriptSource(scriptId string) (string, error) {
	return t.Debugger.GetScriptSource(scriptId)
}

// Returns either an element from our list of ready/known nodeIds or a new un-ready element
// If it's not ready we return false. Note this does have a side effect of adding a potentially
// invalid element to our list of known elements. But it is assumed this method will be called
// with a valid nodeId that chrome has not informed us about yet. Once we are informed, we need
// to update it via our list and not some reference that could disappear.
func (t *Tab) GetElementByNodeId(nodeId int) (*Element, bool) {
	t.eleMutex.RLock()
	ele, ok := t.Elements[nodeId]
	t.eleMutex.RUnlock()
	if ok {
		return ele, true
	}
	newEle := newElement(t, nodeId)
	t.eleMutex.Lock()
	t.Elements[nodeId] = newEle // add non-ready element to our list.
	t.eleMutex.Unlock()
	return newEle, false
}

// Returns the element by searching the top level document for an element with attributeId
// Does not work on frames.
func (t *Tab) GetElementById(attributeId string) (*Element, bool, error) {
	var err error
	var nodeId int
	var doc *gcdapi.DOMNode
	selector := "#" + attributeId
	doc, err = t.GetDocument()
	if err != nil {
		return nil, false, err
	}

	nodeId, err = t.DOM.QuerySelector(doc.NodeId, selector)
	if err != nil {
		return nil, false, err
	}
	ele, ready := t.GetElementByNodeId(nodeId)
	return ele, ready, nil
}

// Get all Elements that match a selector from the top level document
func (t *Tab) GetElementsBySelector(selector string) ([]*Element, error) {
	// get document
	docNode, err := t.GetDocument()
	if err != nil {
		return nil, err
	}

	nodeIds, errQuery := t.DOM.QuerySelectorAll(docNode.NodeId, selector)
	if errQuery != nil {
		return nil, errQuery
	}

	elements := make([]*Element, len(nodeIds))

	for k, nodeId := range nodeIds {
		elements[k], _ = t.GetElementByNodeId(nodeId)
	}

	return elements, nil
}

// Gets all frame ids and urls from the top level document.
func (t *Tab) GetFrameResources() (map[string]string, error) {
	resources, err := t.Page.GetResourceTree()
	if err != nil {
		return nil, err
	}
	resourceMap := make(map[string]string)
	resourceMap[resources.Frame.Id] = resources.Frame.Url
	recursivelyGetFrameResource(resourceMap, resources)
	return resourceMap, nil
}

// Returns the raw source (non-serialized DOM) of the frame. Unfortunately,
// it does not appear possible to get a serialized version using chrome debugger.
// One could extract the urls and load them into a separate tab however.
func (t *Tab) GetFrameSource(id, url string) (string, bool, error) {
	return t.Page.GetResourceContent(id, url)
}

// Issues a left button mousePressed then mouseReleased on the x, y coords provided.
func (t *Tab) Click(x, y int) error {
	// "mousePressed", "mouseReleased", "mouseMoved"
	// enum": ["none", "left", "middle", "right"]
	pressed := "mousePressed"
	released := "mouseReleased"
	modifiers := 0
	timestamp := 0.0
	button := "left"
	clickCount := 1

	if _, err := t.Input.DispatchMouseEvent(pressed, x, y, modifiers, timestamp, button, clickCount); err != nil {
		return err
	}

	if _, err := t.Input.DispatchMouseEvent(released, x, y, modifiers, timestamp, button, clickCount); err != nil {
		return err
	}
	return nil
}

func (t *Tab) SendKeys(text string) error {
	theType := "char"
	modifiers := 0
	timestamp := 0.0
	unmodifiedText := ""
	keyIdentifier := ""
	code := ""
	key := ""
	windowsVirtualKeyCode := 0
	nativeVirtualKeyCode := 0
	autoRepeat := false
	isKeypad := false
	isSystemKey := false

	// loop over input, looking for system keys and handling them
	for _, inputchar := range text {
		input := string(inputchar)

		// check system keys
		switch input {
		case "\r", "\n", "\t", "\b":
			if err := t.pressSystemKey(input); err != nil {
				return err
			}
			continue
		}
		_, err := t.Input.DispatchKeyEvent(theType, modifiers, timestamp, input, unmodifiedText, keyIdentifier, code, key, windowsVirtualKeyCode, nativeVirtualKeyCode, autoRepeat, isKeypad, isSystemKey)
		if err != nil {
			return err
		}
	}
	return nil
}

// Super ghetto, i know.
func (t *Tab) pressSystemKey(systemKey string) error {
	systemKeyCode := 0
	keyIdentifier := ""
	switch systemKey {
	case "\b":
		keyIdentifier = "Backspace"
		systemKeyCode = 8
	case "\t":
		keyIdentifier = "Tab"
		systemKeyCode = 9
	case "\r", "\n":
		systemKey = "\r"
		keyIdentifier = "Enter"
		systemKeyCode = 13
	}

	modifiers := 0
	timestamp := 0.0
	unmodifiedText := ""
	autoRepeat := false
	isKeypad := false
	isSystemKey := false
	if _, err := t.Input.DispatchKeyEvent("rawKeyDown", modifiers, timestamp, systemKey, systemKey, keyIdentifier, keyIdentifier, "", systemKeyCode, systemKeyCode, autoRepeat, isKeypad, isSystemKey); err != nil {
		return err
	}
	if _, err := t.Input.DispatchKeyEvent("char", modifiers, timestamp, systemKey, unmodifiedText, "", "", "", 0, 0, autoRepeat, isKeypad, isSystemKey); err != nil {
		return err
	}
	return nil
}

// Injects custom javascript prior to the page loading on all frames. Returns an identifier which can be
// used to remove the script.
func (t *Tab) InjectScriptOnLoad(scriptSource string) (string, error) {
	scriptId, err := t.Page.AddScriptToEvaluateOnLoad(scriptSource)
	if err != nil {
		return "", err
	}
	return scriptId, nil
}

func (t *Tab) RemoveScriptFromOnLoad(scriptId string) error {
	_, err := t.Page.RemoveScriptToEvaluateOnLoad(scriptId)
	return err
}

// Takes a screenshot of the currently loaded page (only the dimensions visible in browser window)
func (t *Tab) GetScreenShot() ([]byte, error) {
	var imgBytes []byte
	img, err := t.Page.CaptureScreenshot()
	if err != nil {
		return nil, err
	}
	imgBytes, err = base64.StdEncoding.DecodeString(img)
	if err != nil {
		return nil, err
	}
	return imgBytes, nil
}

// Returns the current url of the top level document
func (t *Tab) GetCurrentUrl() (string, error) {
	doc, err := t.GetDocument()
	if err != nil {
		return "", err
	}
	return doc.DocumentURL, nil
}

// Returns the cookies from the tab.
func (t *Tab) GetCookies() ([]*gcdapi.NetworkCookie, error) {
	return t.Page.GetCookies()
}

// Deletes the cookie from the browser
func (t *Tab) DeleteCookie(cookieName, url string) error {
	_, err := t.Page.DeleteCookie(cookieName, url)
	return err
}

// Override the user agent for requests going out.
func (t *Tab) SetUserAgent(userAgent string) error {
	_, err := t.Network.SetUserAgentOverride(userAgent)
	return err
}

// Registers chrome to start retrieving console messages, caller must pass in call back
// function to handle it.
func (t *Tab) GetConsoleMessages(messageHandler ConsoleMessageFunc) {
	t.Subscribe("Console.messageAdded", t.defaultConsoleMessageAdded(messageHandler))
}

// Stops the debugger service from sending console messages and closes the channel
// Pass shouldDisable as true if you wish to disable Console debugger
func (t *Tab) StopConsoleMessages(shouldDisable bool) error {
	var err error
	t.Unsubscribe("Console.messageAdded")
	if shouldDisable {
		_, err = t.Console.Disable()
	}
	return err
}

// Listens to network traffic, either handler can be nil in which case we'll only call the handler defined.
func (t *Tab) ListenNetworkTraffic(requestHandlerFn NetworkRequestHandlerFunc, responseHandlerFn NetworkResponseHandlerFunc) error {
	if requestHandlerFn == nil && responseHandlerFn == nil {
		return nil
	}
	_, err := t.Network.Enable()
	if err != nil {
		return err
	}

	if requestHandlerFn != nil {
		t.Subscribe("Network.requestWillBeSent", func(target *gcd.ChromeTarget, payload []byte) {
			if target != t.ChromeTarget {
				log.Printf("got a request for a different target")
			}
			message := &gcdapi.NetworkRequestWillBeSentEvent{}
			if err := json.Unmarshal(payload, message); err == nil {
				p := message.Params
				request := &NetworkRequest{RequestId: p.RequestId, FrameId: p.FrameId, LoaderId: p.LoaderId, DocumentURL: p.DocumentURL, Request: p.Request, Timestamp: p.Timestamp, Initiator: p.Initiator, RedirectResponse: p.RedirectResponse, Type: p.Type}
				requestHandlerFn(t, request)
			}
		})
	}

	if responseHandlerFn != nil {
		t.Subscribe("Network.responseReceived", func(target *gcd.ChromeTarget, payload []byte) {
			message := &gcdapi.NetworkResponseReceivedEvent{}
			if err := json.Unmarshal(payload, message); err == nil {
				p := message.Params
				response := &NetworkResponse{RequestId: p.RequestId, FrameId: p.FrameId, LoaderId: p.LoaderId, Response: p.Response, Timestamp: p.Timestamp, Type: p.Type}
				responseHandlerFn(t, response)
			}
		})
	}
	return nil
}

// Unsubscribes from network request/response events and disables the Network debugger.
// Pass shouldDisable as true if you wish to disable the network
func (t *Tab) StopListeningNetwork(shouldDisable bool) error {
	var err error
	t.Unsubscribe("Network.requestWillBeSent")
	t.Unsubscribe("Network.responseReceived")
	if shouldDisable {
		_, err = t.Network.Disable()
	}
	return err
}

// Listens for storage events, storageFn should switch on type of cleared, removed, added or updated.
// cleared holds IsLocalStorage and SecurityOrigin values only
// removed contains above plus Key
// added contains above plus NewValue
// updated contains above plus OldValue
func (t *Tab) ListenStorageEvents(storageFn StorageFunc) error {
	_, err := t.DOMStorage.Enable()
	if err != nil {
		return err
	}
	t.Subscribe("Storage.domStorageItemsCleared", func(target *gcd.ChromeTarget, payload []byte) {
		message := &gcdapi.DOMStorageDomStorageItemsClearedEvent{}
		if err := json.Unmarshal(payload, message); err == nil {
			p := message.Params
			storageEvent := &StorageEvent{IsLocalStorage: p.StorageId.IsLocalStorage, SecurityOrigin: p.StorageId.SecurityOrigin}
			storageFn(t, "cleared", storageEvent)
		}
	})
	t.Subscribe("Storage.domStorageItemRemoved", func(target *gcd.ChromeTarget, payload []byte) {
		message := &gcdapi.DOMStorageDomStorageItemRemovedEvent{}
		if err := json.Unmarshal(payload, message); err == nil {
			p := message.Params
			storageEvent := &StorageEvent{IsLocalStorage: p.StorageId.IsLocalStorage, SecurityOrigin: p.StorageId.SecurityOrigin, Key: p.Key}
			storageFn(t, "removed", storageEvent)
		}
	})
	t.Subscribe("Storage.domStorageItemAdded", func(target *gcd.ChromeTarget, payload []byte) {
		message := &gcdapi.DOMStorageDomStorageItemAddedEvent{}
		if err := json.Unmarshal(payload, message); err == nil {
			p := message.Params
			storageEvent := &StorageEvent{IsLocalStorage: p.StorageId.IsLocalStorage, SecurityOrigin: p.StorageId.SecurityOrigin, Key: p.Key, NewValue: p.NewValue}
			storageFn(t, "added", storageEvent)
		}
	})
	t.Subscribe("Storage.domStorageItemUpdated", func(target *gcd.ChromeTarget, payload []byte) {
		message := &gcdapi.DOMStorageDomStorageItemUpdatedEvent{}
		if err := json.Unmarshal(payload, message); err == nil {
			p := message.Params
			storageEvent := &StorageEvent{IsLocalStorage: p.StorageId.IsLocalStorage, SecurityOrigin: p.StorageId.SecurityOrigin, Key: p.Key, NewValue: p.NewValue, OldValue: p.OldValue}
			storageFn(t, "updated", storageEvent)
		}
	})
	return nil
}

// Stops listening for storage events, set shouldDisable to true if you wish to disable DOMStorage debugging.
func (t *Tab) StopListeningStorage(shouldDisable bool) error {
	var err error
	t.Unsubscribe("Storage.domStorageItemsCleared")
	t.Unsubscribe("Storage.domStorageItemRemoved")
	t.Unsubscribe("Storage.domStorageItemAdded")
	t.Unsubscribe("Storage.domStorageItemUpdated")

	if shouldDisable {
		_, err = t.DOMStorage.Disable()
	}
	return err
}

// Set a handler for javascript prompts, most likely you should call tab.Page.HandleJavaScriptDialog(accept bool, msg string)
// to actually handle the prompt, otherwise the tab will be blocked waiting for input and never additional events.
func (t *Tab) SetJavaScriptPromptHandler(promptHandlerFn PromptHandlerFunc) {
	t.Subscribe("Page.javascriptDialogOpening", func(target *gcd.ChromeTarget, payload []byte) {
		log.Printf("Javascript Dialog Opened!: %s\n", string(payload))
		message := &gcdapi.PageJavascriptDialogOpeningEvent{}
		if err := json.Unmarshal(payload, message); err == nil {
			promptHandlerFn(t, message.Params.Message, message.Params.Type)
		}
	})
}

// handles console messages coming in, responds by calling call back function
func (t *Tab) defaultConsoleMessageAdded(fn ConsoleMessageFunc) GcdResponseFunc {
	return func(target *gcd.ChromeTarget, payload []byte) {
		message := &gcdapi.ConsoleMessageAddedEvent{}
		err := json.Unmarshal(payload, message)
		if err == nil {
			// call the callback handler
			fn(t, message.Params.Message)
		}

	}
}

// our default loadFiredEvent handler, returns a response to resp channel to navigate once complete.
func (t *Tab) defaultLoadFired(resp chan<- int) GcdResponseFunc {
	return func(target *gcd.ChromeTarget, payload []byte) {
		target.Unsubscribe("Page.loadEventFired")
		header := &gcdapi.PageLoadEventFiredEvent{}
		err := json.Unmarshal(payload, header)
		if err != nil {
			resp <- -1
		}
		resp <- 0
		close(resp)
	}
}

// Iterate over frame resources and return a map of id => urls
func recursivelyGetFrameResource(resourceMap map[string]string, resource *gcdapi.PageFrameResourceTree) {
	for _, frame := range resource.ChildFrames {
		resourceMap[frame.Frame.Id] = frame.Frame.Url
		recursivelyGetFrameResource(resourceMap, frame)
	}
}

func (t *Tab) subscribeNodeChanges() {
	// see tab_subscribers
	t.subscribeSetChildNodes()
	t.subscribeAttributeModified()
	t.subscribeAttributeRemoved()
	t.subscribeCharacterDataModified()
	t.subscribeChildNodeCountUpdated()
	t.subscribeChildNodeInserted()
	t.subscribeChildNodeRemoved()
	// This doesn't seem useful.
	// t.subscribeInlineStyleInvalidated()
	t.subscribeDocumentUpdated()
}

func (t *Tab) listenNodeChanges() {
	for {
		select {
		case nodeChangeEvent := <-t.nodeChange:
			t.handleNodeChange(nodeChangeEvent)
		}
	}
}

func (t *Tab) handleNodeChange(change *NodeChangeEvent) {
	switch change.EventType {
	case DocumentUpdatedEvent:
		log.Println("documentUpdated, deleting all nodes")
		t.eleMutex.Lock()
		t.Elements = make(map[int]*Element)
		t.eleMutex.Unlock()
	case SetChildNodesEvent:
		log.Println("SetChildNodesEvent, adding new nodes")
		t.createNewNodes(change.Nodes)
	case AttributeModifiedEvent:
		log.Println("attribute modified")
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.updateAttribute(change.Name, change.Value)
		}
	case AttributeRemovedEvent:
		log.Println("attribute removed")
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.removeAttribute(change.Name)
		}
	case CharacterDataModifiedEvent:
		log.Println("character data updated")
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.updateCharacterData(change.CharacterData)
		}
	case ChildNodeCountUpdatedEvent:
		log.Println("character data updated")
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.updateChildNodeCount(change.ChildNodeCount)
		}
	case ChildNodeInsertedEvent:
		log.Println("child node inserted")
		ele := t.knownElement(change.Node)
		t.eleMutex.Lock()
		t.Elements[change.NodeId] = ele
		t.eleMutex.Unlock()
	case ChildNodeRemovedEvent:
		log.Println("child node removed, deleting from list")
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.setInvalidated(true)
		}
		t.eleMutex.Lock()
		delete(t.Elements, change.NodeId)
		t.eleMutex.Unlock()
	}
}

// Called if the element is known about but not yet populated. If it is not
// known, we create a new element. If it is known we populate it and return it.
func (t *Tab) knownElement(node *gcdapi.DOMNode) *Element {
	if ele, ok := t.getElement(node.NodeId); ok {
		ele.populateElement(node)
		return ele
	}
	newEle := newReadyElement(t, node)
	return newEle
}

// recursively create new elements off nodes and their children
// Handle known and unknown elements by calling knownElement prior to
// adding to our list.
func (t *Tab) createNewNodes(nodes []*gcdapi.DOMNode) {
	for _, newNode := range nodes {
		log.Printf("Adding new Node: %s id: %d\n", newNode.NodeName, newNode.NodeId)
		newEle := t.knownElement(newNode)
		t.eleMutex.Lock()
		t.Elements[newNode.NodeId] = newEle
		t.eleMutex.Unlock()
		t.createNewNodes(newNode.Children)
	}
}

// safely returns the element by looking it up by nodeId
func (t *Tab) getElement(nodeId int) (*Element, bool) {
	t.eleMutex.RLock()
	defer t.eleMutex.RUnlock()
	ele, ok := t.Elements[nodeId]
	return ele, ok
}

// Safely adds the nodes in the document to our list of elements
func (t *Tab) addDocumentNodes(doc *gcdapi.DOMNode) {
	newEle := t.knownElement(doc)
	t.eleMutex.Lock()
	t.Elements[doc.NodeId] = newEle
	t.eleMutex.Unlock()
	t.createNewNodes(doc.Children)
}
