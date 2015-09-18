package autogcd

import (
	"encoding/json"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
	"log"
	"sync"
)

// when we are unable to find an element/nodeId
type ElementNotFoundErr struct {
	Message string
}

func (e *ElementNotFoundErr) Error() string {
	return "Unable to find element: " + e.Message
}

// when we are unable to access a tab
type InvalidTabErr struct {
	Message string
}

func (e *InvalidTabErr) Error() string {
	return "Unable to access tab: " + e.Message
}

type GcdResponseFunc func(target *gcd.ChromeTarget, payload []byte)

type ConsoleMessageFunc func(tab *Tab, message *gcdapi.ConsoleConsoleMessage)

type Tab struct {
	*gcd.ChromeTarget
	eleMutex   *sync.RWMutex
	Elements   map[int]*Element
	nodeChange chan *NodeChangeEvent
}

func NewTab(target *gcd.ChromeTarget) *Tab {
	t := &Tab{ChromeTarget: target}
	t.eleMutex = &sync.RWMutex{}
	t.Elements = make(map[int]*Element)
	t.nodeChange = make(chan *NodeChangeEvent)
	t.Page.Enable()
	t.DOM.Enable()
	t.Console.Enable()
	t.Debugger.Enable()
	t.subscribeNodeChanges()
	go t.listenNodeChanges()
	return t
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

// Safely adds the nodes in the document to our list of elements
func (t *Tab) addDocumentNodes(doc *gcdapi.DOMNode) {
	newEle := t.knownElement(doc)
	t.eleMutex.Lock()
	t.Elements[doc.NodeId] = newEle
	t.eleMutex.Unlock()
	t.createNewNodes(doc.Children)
}

// Navigates to a URL and does not return until the Page.loadEventFired event.
// Updates the list of elements
// Returns the frameId of the Tab that this navigation occured on.
func (t *Tab) Navigate(url string) (string, error) {
	//var doc *gcdapi.DOMNode
	resp := make(chan int, 1)

	t.Subscribe("Page.loadEventFired", t.defaultLoadFired(resp))
	frameId, err := t.Page.Navigate(url)
	if err != nil {
		return "", err
	}
	<-resp
	// update the list of elements
	_, err = t.GetDocument()
	if err != nil {
		return "", err
	}
	//_, err = t.DOM.RequestChildNodes(doc.NodeId, -1)
	return frameId, nil
}

// Registers chrome to start retrieving console messages, caller must pass in call back
// function to handle it.
func (t *Tab) GetConsoleMessages(messageHandler ConsoleMessageFunc) {
	t.Subscribe("Console.messageAdded", t.defaultConsoleMessageAdded(messageHandler))
}

// Stops the debugger service from sending console messages and closes the channel
func (t *Tab) StopConsoleMessages() {
	t.Unsubscribe("Console.messageAdded")
}

// Returns the top window documents source, as visible
func (t *Tab) GetPageSource() (string, error) {
	node, err := t.GetDocument()
	if err != nil {
		return "", err
	}
	return t.DOM.GetOuterHTML(node.NodeId)
}

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
