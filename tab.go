package autogcd

import (
	"encoding/json"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdprotogen/types"
	"log"
)

type InvalidTabErr struct {
	Message string
}

func (e *InvalidTabErr) Error() string {
	return "Unable to access tab: " + e.Message
}

type GcdResponse func(target *gcd.ChromeTarget, payload []byte)

type Tab struct {
	*gcd.ChromeTarget
}

func NewTab(target *gcd.ChromeTarget) *Tab {
	t := &Tab{ChromeTarget: target}
	t.Page().Enable()
	t.DOM().Enable()
	return t
}

// Navigates to a URL and does not return until the Page.loadEventFired event.
// Returns the frameId of the Tab that this navigation occured on.
func (t *Tab) Navigate(url string) (string, error) {
	resp := make(chan int, 1)
	t.Subscribe("Page.loadEventFired", t.defaultLoadFired(resp))
	frameId, err := t.Page().Navigate(url)
	if err != nil {
		return "", err
	}
	<-resp
	return string(*frameId), nil
}

// Registers chrome to start retrieving console messages, caller must pass in a channel
// to get notified when events come in. Caller should call StopConsoleMessages and close
// the channel.
func (t *Tab) GetConsoleMessages() chan types.ChromeConsoleConsoleMessage {
	messageCh := make(chan types.ChromeConsoleConsoleMessage)
	t.Console().Enable()
	t.Subscribe("Console.messageAdded", t.defaultConsoleMessageAdded(messageCh))
	return messageCh
}

// Stops the debugger service from sending console messages and closes the channel
func (t *Tab) StopConsoleMessages(messageCh chan types.ChromeConsoleConsoleMessage) {
	t.Console().Disable()
	close(messageCh)
}

// Returns the top window documents source, as visible
func (t *Tab) GetPageSource() (string, error) {
	var node *types.ChromeDOMNode
	var err error
	node, err = t.DOM().GetDocument()
	if err != nil {
		return "", err
	}
	return t.DOM().GetOuterHTML(node.NodeId)
}

func (t *Tab) GetElementsBySelector(selector string) ([]*Element, error) {
	nodeIds, err := t.GetNodeIdsBySelector(selector)
	if err != nil {
		return err
	}
	elements := make([]*Element, len(nodeIds))
	for i := 0; i < len(nodeIds); i++ {
		elements[i] = newElement(t, nodeIds[i])
	}
	return elements
}

func (t *Tab) GetNodeIdsBySelector(selector string) ([]int, error) {
	docNode, err := t.DOM().GetDocument()
	if err != nil {
		return nil, err
	}

	nodeIds, errQuery := t.DOM().QuerySelectorAll(docNode.NodeId, selector)
	if errQuery != nil {
		return nil, errQuery
	}

	nodes := make([]int, len(nodeIds))
	for k, v := range nodeIds {
		nodes[k] = int(*v)
	}
	return nodes, nil
}

// Gets all frame ids and urls from the top level document.
func (t *Tab) GetFrameResources() (map[string]string, error) {
	resources, err := t.Page().GetResourceTree()
	if err != nil {
		return nil, err
	}
	resourceMap := make(map[string]string)
	resourceMap[resources.Frame.Id] = resources.Frame.Url
	recursivelyGetFrameResource(resourceMap, resources)
	return resourceMap, nil
}

func recursivelyGetFrameResource(resourceMap map[string]string, resource *types.ChromePageFrameResourceTree) {
	for _, frame := range resource.ChildFrames {
		resourceMap[frame.Frame.Id] = frame.Frame.Url
		recursivelyGetFrameResource(resourceMap, frame)
	}
}

// Returns the raw source (non-serialized DOM) of the frame. Unfortunately,
// it does not appear possible to get a serialized version using chrome debugger.
// One could extract the urls and load them into a separate tab however.
func (t *Tab) GetFrameSource(id, url string) (string, bool, error) {
	nodeId := types.ChromePageFrameId(id)
	return t.Page().GetResourceContent(&nodeId, url)
}

// Returns the outer HTML of the node
func (t *Tab) GetElementSource(id int) (string, error) {
	nodeId := types.ChromeDOMNodeId(id)
	return t.DOM().GetOuterHTML(&nodeId)
}

func (t *Tab) Click(x, y int) error {
	// "mousePressed", "mouseReleased", "mouseMoved"
	// enum": ["none", "left", "middle", "right"]
	pressed := "mousePressed"
	released := "mouseReleased"
	modifiers := 0
	timestamp := 0
	button := "left"
	clickCount := 1

	if _, err := t.Input().DispatchMouseEvent(pressed, x, y, modifiers, timestamp, button, clickCount); err != nil {
		return err
	}

	if _, err := t.Input().DispatchMouseEvent(released, x, y, modifiers, timestamp, button, clickCount); err != nil {
		return err
	}
}

func (t *Tab) GetElementById(int id) (*Element, error) {
	return newElement(t, id)
}

//
func (t *Tab) defaultConsoleMessageAdded(message chan<- types.ChromeConsoleConsoleMessage) GcdResponse {
	return func(target *gcd.ChromeTarget, payload []byte) {
		consoleMessage := &ConsoleEventHeader{}
		err := json.Unmarshal(payload, consoleMessage)
		if err != nil {
			message <- types.ChromeConsoleConsoleMessage{}
		}
		message <- *consoleMessage.Params.Message
	}
}

func (t *Tab) defaultLoadFired(resp chan<- int) GcdResponse {
	return func(target *gcd.ChromeTarget, payload []byte) {
		log.Printf("event fired")
		fired := &PageLoadEventFired{}
		err := json.Unmarshal(payload, fired)
		if err != nil {
			resp <- -1
		}
		resp <- fired.timestamp
	}
}
