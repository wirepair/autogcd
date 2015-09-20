package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
	"strings"
	"time"
)

type InvalidElementErr struct {
}

func (e *InvalidElementErr) Error() string {
	return "this element has been invalidated"
}

// for when we have an element that has not been populated
// with data yet.
type ElementNotReadyErr struct {
}

func (e *ElementNotReadyErr) Error() string {
	return "this element is not ready"
}

type InvalidDimensionsErr struct {
	Message string
}

func (e *InvalidDimensionsErr) Error() string {
	return "invalid dimensions " + e.Message
}

// An abstraction over a DOM element, it can be in three modes
// NotReady - it's data has not been returned to us by the debugger yet.
// Ready - the debugger has given us the DOMNode reference.
// Invalidated - The Element has been destroyed.
// Certain actions require that the Element be populated (getting nodename/type)
// If you need this information, wait for IsReady() to return true
type Element struct {
	tab            *Tab              // reference to the containing tab
	node           *gcdapi.DOMNode   // the dom node, taken from the document
	attributes     map[string]string // dom attributes
	nodeName       string
	characterData  string
	childNodeCount int
	nodeType       int
	readyGate      chan struct{}
	id             int  // nodeId in chrome
	ready          bool // has this elements data been populated by setChildNodes or GetDocument?
	invalidated    bool // has this node been invalidated (removed?)
}

func newElement(tab *Tab, nodeId int) *Element {
	e := &Element{}
	e.tab = tab
	e.attributes = make(map[string]string)
	e.readyGate = make(chan struct{})
	e.id = nodeId
	return e
}

func newReadyElement(tab *Tab, node *gcdapi.DOMNode) *Element {
	e := &Element{}
	e.tab = tab
	e.attributes = make(map[string]string)
	e.readyGate = make(chan struct{})
	e.nodeName = strings.ToLower(node.NodeName)
	e.id = node.NodeId
	e.populateElement(node)
	return e
}

// Has the Chrome Debugger notified us of this Elements data yet?
func (e *Element) IsReady() bool {
	return (e.ready && !e.invalidated)
}

// Has the debugger invalidated (removed) the element from the DOM?
func (e *Element) IsInvalid() bool {
	return e.invalidated
}

// Is this element a frame?
func (e *Element) IsFrame() bool {
	return e.node.FrameId != ""
}

// Returns the frameId if IsFrame is true and the Element is in a Ready state.
func (e *Element) FrameId() string {
	return e.node.FrameId
}

// populate the Element with node data.
func (e *Element) populateElement(node *gcdapi.DOMNode) {
	e.node = node
	e.nodeType = node.NodeType
	e.nodeName = strings.ToLower(node.NodeName)
	e.childNodeCount = node.ChildNodeCount
	// close it
	if !e.ready {
		close(e.readyGate)
	}
	e.ready = true
}

// updates the attribute name/value pair
func (e *Element) updateAttribute(name, value string) {
	e.attributes[name] = value
}

// removes the attribute from our attributes list.
func (e *Element) removeAttribute(name string) {
	delete(e.attributes, name)
}

// updates character data
func (e *Element) updateCharacterData(newValue string) {
	e.characterData = newValue
}

// updates child node counts.
func (e *Element) updateChildNodeCount(newValue int) {
	e.childNodeCount = newValue
}

// The element has become invalid.
func (e *Element) setInvalidated(invalid bool) {
	e.invalidated = invalid
}

// Returns the tag name (input, div) if the element is in a ready state.
func (e *Element) GetTagName() (string, error) {
	if !e.ready {
		return "", &ElementNotReadyErr{}
	}
	return e.nodeName, nil
}

// Returns the node type if the element is in a ready state.
func (e *Element) GetNodeType() (int, error) {
	if !e.ready {
		return -1, &ElementNotReadyErr{}
	}
	return e.nodeType, nil
}

// Returns the CSS Style Text of the element, returns the inline style first
// and the attribute style second, or error.
func (e *Element) GetCssStyleText() (string, string, error) {
	inline, attribute, err := e.tab.CSS.GetInlineStylesForNode(e.id)
	if err != nil {
		return "", "", err
	}
	return inline.CssText, attribute.CssText, nil
}

// Returns event listeners for the element, both static and dynamically bound.
func (e *Element) GetEventListeners() ([]*gcdapi.DOMDebuggerEventListener, error) {
	rro, err := e.tab.DOM.ResolveNode(e.id, "")
	if err != nil {
		return nil, err
	}
	eventListeners, err := e.tab.DOMDebugger.GetEventListeners(rro.ObjectId)
	if err != nil {
		return nil, err
	}
	return eventListeners, nil
}

// If we are ready, just return, if we are not, wait for the readyGate
// to be closed or for the timeout timer to fire.
func (e *Element) WaitForReady() error {
	if e.ready {
		return nil
	}

	timeout := time.NewTimer(e.tab.elementTimeout * time.Second)
	select {
	case <-e.readyGate:
		return nil
	case <-timeout.C:
		return &ElementNotReadyErr{}
	}
}

// Get attributes of the node in sequential name,value pairs in the slice.
func (e *Element) GetAttributes() (map[string]string, error) {
	attr, err := e.tab.DOM.GetAttributes(e.id)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(attr); i += 2 {
		e.updateAttribute(attr[i], attr[i+1])
	}
	return e.attributes, nil
}

// Works like WebDriver's clear(), simply sets the attribute value for input
// or clears the value for textarea. This element must be ready so we can
// properly read the nodeName value.
func (e *Element) Clear() error {
	var err error

	if !e.ready {
		return &ElementNotReadyErr{}
	}

	if e.nodeName == "textarea" {
		_, err = e.tab.DOM.SetNodeValue(e.id, "")
	}
	if e.nodeName == "input" {
		_, err = e.tab.DOM.SetAttributeValue(e.id, "value", "")
	}
	return err
}

// Returns the outer html of the element.
func (e *Element) GetSource() (string, error) {
	if e.invalidated {
		return "", &InvalidElementErr{}
	}
	return e.tab.DOM.GetOuterHTML(e.id)
}

// Clicks the element in the center of the element.
func (e *Element) Click() error {
	var x int
	var y int

	points, err := e.Dimensions()
	if err != nil {
		return err
	}

	x, y, err = centroid(points)
	if err != nil {
		return err
	}
	// click the centroid of the element.
	return e.tab.Click(x, y)
}

// SendKeys - sends each individual character after focusing (clicking) on the element.
// Extremely basic, doesn't take into account most/all system keys except enter.
func (e *Element) SendKeys(text string) error {
	err := e.Click()
	if err != nil {
		return err
	}
	return e.tab.SendKeys(text)
}

// Returns the dimensions of the element.
func (e *Element) Dimensions() ([]float64, error) {
	var points []float64
	box, err := e.tab.DOM.GetBoxModel(e.id)
	if err != nil {
		return nil, err
	}
	points = box.Content
	return points, nil
}

// finds the centroid of an arbitrary number of points.
// Assumes points[i] = x, points[i+1] = y.
func centroid(points []float64) (int, int, error) {
	pointLen := len(points)
	if pointLen%2 != 0 {
		return -1, -1, &InvalidDimensionsErr{"number of points are not divisible by two"}
	}
	x := 0
	y := 0
	for i := 0; i < pointLen; i = i + 2 {
		x += int(points[i])
		y += int(points[i+1])
	}
	return x / (pointLen / 2), y / (pointLen / 2), nil
}
