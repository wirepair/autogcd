package autogcd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	return "Unable to find element " + e.Message
}

// When we are unable to access a tab
type InvalidTabErr struct {
	Message string
}

func (e *InvalidTabErr) Error() string {
	return "Unable to access tab: " + e.Message
}

// Returned when an injected script caused an error
type ScriptEvaluationErr struct {
	Message          string
	ExceptionText    string
	ExceptionDetails *gcdapi.DebuggerExceptionDetails
}

func (e *ScriptEvaluationErr) Error() string {
	return e.Message + " " + e.ExceptionText
}

// When Tab.Navigate has timed out
type TimeoutErr struct {
	Message string
}

func (e *TimeoutErr) Error() string {
	return "Timed out attempting to " + e.Message
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

// A function to listen for DOM Node Change Events
type DomChangeHandlerFunc func(tab *Tab, change *NodeChangeEvent)

// Our tab object for driving a specific tab and gathering elements.
type Tab struct {
	*gcd.ChromeTarget                        // underlying chrometarget
	eleMutex           *sync.RWMutex         // locks our elements when added/removed.
	elements           map[int]*Element      // our map of elements for this tab
	topNodeId          int                   // the nodeId of the current top level #document
	topFrameId         string                // the frameId of the current top level #document
	isNavigating       bool                  // are we currently navigating (between Page.Navigate -> page.loadEventFired)
	isTransitioning    bool                  // has navigation occurred on the top frame (not due to Navigate() being called)
	nodeChange         chan *NodeChangeEvent // for receiving node change events from tab_subscribers
	navigationCh       chan int              // for receiving navigation complete messages while isNavigating is true
	setNodesWg         *sync.WaitGroup       // tracks setChildNode event calls when Navigating.
	navigationTimeout  time.Duration         // (seconds) amount of time to wait before failing navigation
	elementTimeout     time.Duration         // (seconds) amount of time to wait for element readiness
	stabilityTimeout   time.Duration         // (seconds) amount of time to give up waiting for stability
	stabilityTime      time.Duration         // (milliseconds) amount of time of no activity to consider the DOM stable
	lastNodeChangeTime time.Time             // timestamp of when the last node change occurred
	domChangeHandler   DomChangeHandlerFunc  // allows the caller to be notified of DOM change events.
}

// Creates a new tab using the underlying ChromeTarget
func NewTab(target *gcd.ChromeTarget) *Tab {
	t := &Tab{ChromeTarget: target}
	t.eleMutex = &sync.RWMutex{}
	t.elements = make(map[int]*Element)
	t.nodeChange = make(chan *NodeChangeEvent)
	t.navigationTimeout = 30           // default 30 seconds for timeout
	t.isNavigating = false             // for when Tab.Navigate() is called
	t.isTransitioning = false          // for when an action or redirect causes the top level frame to navigate
	t.navigationCh = make(chan int, 1) // for signaling navigation complete
	t.setNodesWg = &sync.WaitGroup{}   // wait for setChildNode events to complete
	t.elementTimeout = 5               // default 5 seconds for waiting for element.
	t.stabilityTimeout = 2             // default 2 seconds before we give up waiting for stability
	t.stabilityTime = 300              // default 300 ms for considering the DOM stable
	t.domChangeHandler = nil
	t.Page.Enable()
	t.DOM.Enable()
	t.Console.Enable()
	t.Debugger.Enable()
	t.subscribeEvents()
	go t.listenDOMChanges()
	return t
}

// How long to wait in seconds for navigations before giving up
func (t *Tab) SetNavigationTimeout(timeout time.Duration) {
	t.navigationTimeout = timeout
}

// How long to wait in seconds for ele.WaitForReady() before giving up
func (t *Tab) SetElementWaitTimeout(timeout time.Duration) {
	t.elementTimeout = timeout
}

// How long to wait in milliseconds for WaitStable() to return
func (t *Tab) SetStabilityTimeout(timeout time.Duration) {
	t.stabilityTimeout = timeout
}

// How long to wait for no node changes before we consider the DOM stable
// note that stability timeout will fire if the DOM is constantly changing.
// stabilityTime is in milliseconds, default is 300 ms.
func (t *Tab) SetStabilityTime(stabilityTime time.Duration) {
	t.stabilityTime = stabilityTime
}

// Navigates to a URL and does not return until the Page.loadEventFired event
// as well as all setChildNode events have completed.
// Returns the frameId of the Tab that this navigation occured on.
func (t *Tab) Navigate(url string) (string, error) {
	t.isNavigating = true
	frameId, err := t.Page.Navigate(url)
	timeoutTimer := time.NewTimer(t.navigationTimeout * time.Second)
	if err != nil {
		return "", err
	}

	select {
	case <-t.navigationCh:
		timeoutTimer.Stop()
	case <-timeoutTimer.C:
		return "", &TimeoutErr{Message: "navigate to: " + url}
	}
	t.isNavigating = false

	// update the list of elements
	_, err = t.GetDocument()
	if err != nil {
		return "", err
	}
	// wait for all setChildNode event processing to finish so we have a valid
	// set of elements
	t.setNodesWg.Wait()
	return frameId, nil
}

// Call this after clicking on elements or sending keys to see if we are transitioning to
// a new page. Pass in a delay to give the browser some time to handle the click/sendKeys event
// before starting our transitionin check. Then we will check if we are transitioning every 20 ms,
// give up after navigationTimeout. Pass true for waitStable to wait for dom node changes to finish
func (t *Tab) WaitTransitioning(delay time.Duration, waitStable bool) error {
	checkRate := time.Duration(20)
	timeoutTimer := time.NewTimer(t.navigationTimeout * time.Second)
	transitionCheck := time.NewTicker(checkRate * time.Millisecond) // check last node change every 20 seconds
	// give the Tab a bit of time to wait before we start checking for navigation events
	delayTimer := time.NewTimer(delay * time.Millisecond)
	<-delayTimer.C

	for {
		select {
		case <-transitionCheck.C:
			// done transitioning,
			if t.isTransitioning == false {
				// kill our timeoutTimer and start stability check if waitStable is true
				timeoutTimer.Stop()
				if waitStable {
					return t.WaitStable()
				}
				return nil
			}
		case <-timeoutTimer.C:
			return &TimeoutErr{Message: "transitioning failed due to timeout"}
		}
	}
	return nil
}

// A very rudementary stability check, compare current time with lastNodeChangeTime and see if it
// is greater than the stabilityTime. If it is, that means we haven't seen any activity over the minimum
// allowed time, in which case we consider the DOM stable. Note this will most likely not work for sites
// that insert and remove elements on timer/intervals as it will constantly update our lastNodeChangeTime
// value. However, for most cases this should be enough. This should only be necessary to call when
// a navigation event occurs under the page's control (not a direct tab.Navigate) call. Common examples
// would be submitting an XHR based form that does a history.pushState and does *not* actuall load a new
// page but simply inserts and removes elements dynamically. Returns error only if we timed out.
func (t *Tab) WaitStable() error {
	checkRate := time.Duration(20)
	timeoutTimer := time.NewTimer(t.stabilityTimeout * time.Second)
	if t.stabilityTime < checkRate {
		checkRate = t.stabilityTime / 2 // halve the checkRate of the user supplied stabilityTime

	}
	stableCheck := time.NewTicker(checkRate * time.Millisecond) // check last node change every 20 seconds
	for {
		select {
		case <-timeoutTimer.C:
			return &TimeoutErr{Message: "timed out waiting for DOM stability"}
		case <-stableCheck.C:
			//log.Printf("stable check tick, lastnode change time %v", time.Now().Sub(t.lastNodeChangeTime))
			if time.Now().Sub(t.lastNodeChangeTime) >= t.stabilityTime*time.Millisecond {
				//log.Printf("times up!")
				return nil
			}

		}
	}
	return nil
}

// Returns the source of a script by its scriptId.
func (t *Tab) GetScriptSource(scriptId string) (string, error) {
	return t.Debugger.GetScriptSource(scriptId)
}

// Gets the top document and updates our list of elements in the event they've changed.
func (t *Tab) GetDocument() (*Element, error) {
	doc, err := t.DOM.GetDocument()
	if err != nil {
		return nil, err
	}
	t.topNodeId = doc.NodeId
	t.topFrameId = doc.FrameId
	t.addNodes(doc)
	eleDoc, _ := t.getElement(doc.NodeId)
	return eleDoc, nil
}

// Returns either an element from our list of ready/known nodeIds or a new un-ready element
// If it's not ready we return false. Note this does have a side effect of adding a potentially
// invalid element to our list of known elements. But it is assumed this method will be called
// with a valid nodeId that chrome has not informed us about yet. Once we are informed, we need
// to update it via our list and not some reference that could disappear.
func (t *Tab) GetElementByNodeId(nodeId int) (*Element, bool) {
	t.eleMutex.RLock()
	ele, ok := t.elements[nodeId]
	t.eleMutex.RUnlock()
	if ok {
		return ele, true
	}
	newEle := newElement(t, nodeId)
	t.eleMutex.Lock()
	t.elements[nodeId] = newEle // add non-ready element to our list.
	t.eleMutex.Unlock()
	return newEle, false
}

// Returns the element by searching the top level document for an element with attributeId
// Does not work on frames.
func (t *Tab) GetElementById(attributeId string) (*Element, bool, error) {
	return t.GetDocumentElementById(t.topNodeId, attributeId)
}

// Returns an element from a specific Document.
func (t *Tab) GetDocumentElementById(docNodeId int, attributeId string) (*Element, bool, error) {
	var err error

	docNode, ok := t.getElement(docNodeId)
	if !ok {
		return nil, false, &ElementNotFoundErr{Message: fmt.Sprintf("docNodeId %s not found", docNodeId)}
	}

	selector := "#" + attributeId

	nodeId, err := t.DOM.QuerySelector(docNode.id, selector)
	if err != nil {
		return nil, false, err
	}
	ele, ready := t.GetElementByNodeId(nodeId)
	return ele, ready, nil
}

// Get all elements that match a selector from the top level document
func (t *Tab) GetElementsBySelector(selector string) ([]*Element, error) {
	return t.GetDocumentElementsBySelector(t.topNodeId, selector)
}

// Get all elements that match a selector from the provided document node id.
func (t *Tab) GetDocumentElementsBySelector(docNodeId int, selector string) ([]*Element, error) {
	docNode, ok := t.getElement(docNodeId)
	if !ok {
		return nil, &ElementNotFoundErr{Message: fmt.Sprintf("docNodeId %s not found", docNodeId)}
	}
	nodeIds, errQuery := t.DOM.QuerySelectorAll(docNode.id, selector)
	if errQuery != nil {
		return nil, errQuery
	}

	elements := make([]*Element, len(nodeIds))

	for k, nodeId := range nodeIds {
		elements[k], _ = t.GetElementByNodeId(nodeId)
	}

	return elements, nil
}

// Returns the documents source, as visible, if docId is 0, returns top document source.
func (t *Tab) GetPageSource(docNodeId int) (string, error) {
	if docNodeId == 0 {
		docNodeId = t.topNodeId
	}
	doc, ok := t.getElement(docNodeId)
	if !ok {
		return "", &ElementNotFoundErr{Message: fmt.Sprintf("docNodeId %d not found", docNodeId)}
	}
	return t.DOM.GetOuterHTML(doc.id)
}

// Returns the current url of the top level document
func (t *Tab) GetCurrentUrl() (string, error) {
	return t.GetDocumentCurrentUrl(t.topNodeId)
}

// Returns the current url of the provided docNodeId
func (t *Tab) GetDocumentCurrentUrl(docNodeId int) (string, error) {
	docNode, ok := t.getElement(docNodeId)
	if !ok {
		return "", &ElementNotFoundErr{Message: fmt.Sprintf("docNodeId %s not found", docNodeId)}
	}
	return docNode.node.DocumentURL, nil
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

// Sends keystrokes to whatever is focused, best called from Element.SendKeys which will
// try to focus on the element first. Use \n for Enter, \b for backspace or \t for Tab.
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

// Injects custom javascript prior to the page loading on all frames. Returns scriptId which can be
// used to remove the script. Since this loads on all frames, if you only want the script to interact with the
// top document, you'll need to do checks in the injected script such as testing location.href.
// Alternatively, you can use Tab.EvaluateScript to only work on the global context.
func (t *Tab) InjectScriptOnLoad(scriptSource string) (string, error) {
	scriptId, err := t.Page.AddScriptToEvaluateOnLoad(scriptSource)
	if err != nil {
		return "", err
	}
	return scriptId, nil
}

// Removes the script by the scriptId.
func (t *Tab) RemoveScriptFromOnLoad(scriptId string) error {
	_, err := t.Page.RemoveScriptToEvaluateOnLoad(scriptId)
	return err
}

// Evaluates script in the global context.
func (t *Tab) EvaluateScript(scriptSource string) (*gcdapi.RuntimeRemoteObject, error) {
	objectGroup := "autogcd"
	includeCommandLineAPI := true
	doNotPauseOnExceptionsAndMuteConsole := true
	contextId := 0
	returnByValue := true
	generatePreview := true
	rro, thrown, exception, err := overridenRuntimeEvaluate(t.ChromeTarget, scriptSource, objectGroup, includeCommandLineAPI, doNotPauseOnExceptionsAndMuteConsole, contextId, returnByValue, generatePreview)
	if err != nil {
		return nil, err
	}
	if thrown || exception != nil {
		return nil, &ScriptEvaluationErr{Message: "error executing script: ", ExceptionText: exception.Text, ExceptionDetails: exception}
	}
	return rro, nil
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

// Returns the top document title
func (t *Tab) GetTitle() (string, error) {
	resp, err := t.EvaluateScript("window.top.document.title")
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

// Returns the raw source (non-serialized DOM) of the frame. If you want the visible
// source, call GetPageSource, passing in the frame's nodeId. Make sure you wait for
// the element's WaitForReady() to return without error first.
func (t *Tab) GetFrameSource(frameId, url string) (string, bool, error) {
	return t.Page.GetResourceContent(frameId, url)
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

// Iterate over frame resources and return a map of id => urls
func recursivelyGetFrameResource(resourceMap map[string]string, resource *gcdapi.PageFrameResourceTree) {
	for _, frame := range resource.ChildFrames {
		resourceMap[frame.Frame.Id] = frame.Frame.Url
		recursivelyGetFrameResource(resourceMap, frame)
	}
}

// Returns all documents as elements that are known.
func (t *Tab) GetFrameDocuments() []*Element {
	frames := make([]*Element, 0)
	t.eleMutex.RLock()
	for _, ele := range t.elements {
		if ok, _ := ele.IsDocument(); ok {
			frames = append(frames, ele)
		}
	}
	t.eleMutex.RUnlock()
	return frames
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
// cleared holds IsLocalStorage and SecurityOrigin values only.
// removed contains above plus Key.
// added contains above plus NewValue.
// updated contains above plus OldValue.
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

// Allow the caller to be notified of DOM NodeChangeEvents
func (t *Tab) SetDomChangeHandler(domHandlerFn DomChangeHandlerFunc) {
	t.domChangeHandler = domHandlerFn
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

// see tab_subscribers.go
func (t *Tab) subscribeEvents() {
	// DOM Related

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

	// Navigation Related
	t.subscribeLoadEvent()
	t.subscribeFrameLoadingEvent()
	t.subscribeFrameFinishedEvent()
}

// listens for NodeChangeEvents and dispatches them accordingly. Calls the user
// defined domChangeHandler if bound. Updates the lastNodeChangeTime to the current
// time.
func (t *Tab) listenDOMChanges() {
	for {
		select {
		case nodeChangeEvent := <-t.nodeChange:
			t.handleNodeChange(nodeChangeEvent)
			// if the caller registered a dom change listener, call it
			if t.domChangeHandler != nil {
				t.domChangeHandler(t, nodeChangeEvent)
			}
			t.lastNodeChangeTime = time.Now()
		}
	}
}

// handle node change events, updating, inserting invalidating and removing
func (t *Tab) handleNodeChange(change *NodeChangeEvent) {
	switch change.EventType {
	case DocumentUpdatedEvent:
		// set all elements as invalid and destroy the Elements map.
		t.eleMutex.Lock()
		for _, ele := range t.elements {
			ele.setInvalidated(true)
		}
		t.elements = make(map[int]*Element)
		t.eleMutex.Unlock()
		t.documentUpdated()
	case SetChildNodesEvent:
		t.setNodesWg.Add(1)
		go t.handleSetChildNodes(change.ParentNodeId, change.Nodes)
	case AttributeModifiedEvent:
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.updateAttribute(change.Name, change.Value)
		}
	case AttributeRemovedEvent:
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.removeAttribute(change.Name)
		}
	case CharacterDataModifiedEvent:
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.updateCharacterData(change.CharacterData)
		}
	case ChildNodeCountUpdatedEvent:
		if ele, ok := t.getElement(change.NodeId); ok {
			ele.updateChildNodeCount(change.ChildNodeCount)
			// request the child nodes
			t.requestChildNodes(change.NodeId, -1)
		}
	case ChildNodeInsertedEvent:
		t.handleChildNodeInserted(change.ParentNodeId, change.Node)
	case ChildNodeRemovedEvent:
		t.handleChildNodeRemoved(change.ParentNodeId, change.NodeId)
	}

}

// setChildNode event handling will add nodes to our elements map and update
// the parent reference Children
func (t *Tab) handleSetChildNodes(parentNodeId int, nodes []*gcdapi.DOMNode) {
	for _, node := range nodes {
		t.addNodes(node)
	}
	parent, ready := t.getElement(parentNodeId)
	if ready {
		parent.addChildren(nodes)
	}
	t.lastNodeChangeTime = time.Now()
	t.setNodesWg.Done()
}

// update parent with new child node and add nodes.
func (t *Tab) handleChildNodeInserted(parentNodeId int, node *gcdapi.DOMNode) {
	//log.Printf("child node inserted: id: %d\n", node.NodeId)
	t.addNodes(node)

	parent, ready := t.getElement(parentNodeId)
	if ready {
		parent.addChild(node)
	}
	t.lastNodeChangeTime = time.Now()
}

// update ParentNodeId to remove child and iterate over Children recursively and invalidate them.
func (t *Tab) handleChildNodeRemoved(parentNodeId, nodeId int) {
	//log.Printf("child node REMOVED: %d\n", nodeId)
	ele, ok := t.getElement(nodeId)
	if !ok {
		return
	}
	ele.setInvalidated(true)
	parent, ready := t.getElement(parentNodeId)
	if ready {
		parent.removeChild(ele.node)
	}
	t.invalidateChildren(ele.node)

	t.eleMutex.Lock()
	delete(t.elements, nodeId)
	t.eleMutex.Unlock()
}

// when a childNodeRemoved event occurs, we need to set each child
// to invalidated and remove it from our elements map.
func (t *Tab) invalidateChildren(node *gcdapi.DOMNode) {
	// invalidate & remove ContentDocument node and children
	if node.ContentDocument != nil {
		ele, ok := t.getElement(node.ContentDocument.NodeId)
		if ok {
			t.invalidateRemove(ele)
			t.invalidateChildren(node.ContentDocument)
		}
	}

	if node.Children == nil {
		return
	}

	// invalidate node.Children
	for _, child := range node.Children {
		ele, ok := t.getElement(child.NodeId)
		if !ok {
			continue
		}
		t.invalidateRemove(ele)
		// recurse and remove children of this node
		t.invalidateChildren(ele.node)
	}
}

// Sets the element as invalid and removes it from our elements map
func (t *Tab) invalidateRemove(ele *Element) {
	//log.Printf("invalidating nodeId: %d\n", ele.id)
	ele.setInvalidated(true)
	t.eleMutex.Lock()
	delete(t.elements, ele.id)
	t.eleMutex.Unlock()
}

// the entire document has been invalidated, request all nodes again
func (t *Tab) documentUpdated() {
	log.Printf("document updated, refreshed")
	t.GetDocument()
}

// Ask the debugger service for child nodes
func (t *Tab) requestChildNodes(nodeId, depth int) {
	_, err := t.DOM.RequestChildNodes(nodeId, depth)
	if err != nil {
		log.Printf("error requesting child nodes: %s\n", err)
	}
}

// Called if the element is known about but not yet populated. If it is not
// known, we create a new element. If it is known we populate it and return it.
func (t *Tab) nodeToElement(node *gcdapi.DOMNode) *Element {
	if ele, ok := t.getElement(node.NodeId); ok {
		ele.populateElement(node)
		return ele
	}
	newEle := newReadyElement(t, node)
	return newEle
}

// safely returns the element by looking it up by nodeId
func (t *Tab) getElement(nodeId int) (*Element, bool) {
	t.eleMutex.RLock()
	defer t.eleMutex.RUnlock()
	ele, ok := t.elements[nodeId]
	return ele, ok
}

// Safely adds the nodes in the document to our list of elements
// iterates over children and contentdocuments (if they exist)
// Calls requestchild nodes for each node so we can receive setChildNode
// events for even more nodes
func (t *Tab) addNodes(node *gcdapi.DOMNode) {
	newEle := t.nodeToElement(node)
	t.eleMutex.Lock()
	t.elements[newEle.id] = newEle
	t.eleMutex.Unlock()
	//log.Printf("Added new element: %s\n", newEle)
	t.requestChildNodes(newEle.id, -1)
	if node.Children != nil {
		// add child nodes
		for _, v := range node.Children {
			t.addNodes(v)
		}
	}
	if node.ContentDocument != nil {
		t.addNodes(node.ContentDocument)
	}
	t.lastNodeChangeTime = time.Now()
}
