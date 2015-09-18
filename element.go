package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
	"strings"
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
// NotReady - it's data has not been returned to us by the debugger yet
// Ready - the debugger has given us the DOMNode reference
// Invalidated - The Element has been destroyed
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

func (e *Element) IsReady() bool {
	return (e.ready && !e.invalidated)
}

func (e *Element) IsInvalid() bool {
	return e.invalidated
}

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

func (e *Element) updateAttribute(name, value string) {
	e.attributes[name] = value
}

func (e *Element) removeAttribute(name string) {
	delete(e.attributes, name)
}

func (e *Element) updateCharacterData(newValue string) {
	e.characterData = newValue
}

func (e *Element) updateChildNodeCount(newValue int) {
	e.childNodeCount = newValue
}

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
// to be closed
func (e *Element) WaitForReady() {
	if e.ready {
		return
	}
	<-e.readyGate
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

// Works like WebDriver's clear(), simply sets the attribute value for input.
// or clears the value for textarea. This element must be ready so we can
// properly read the nodeName value
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
			if err := e.pressSystemKey(input); err != nil {
				return err
			}
			continue
		}
		_, err = e.tab.Input.DispatchKeyEvent(theType, modifiers, timestamp, input, unmodifiedText, keyIdentifier, code, key, windowsVirtualKeyCode, nativeVirtualKeyCode, autoRepeat, isKeypad, isSystemKey)
		if err != nil {
			return err
		}
	}
	return err
}

// Super ghetto, i know.
func (e *Element) pressSystemKey(systemKey string) error {
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
	if _, err := e.tab.Input.DispatchKeyEvent("rawKeyDown", modifiers, timestamp, systemKey, systemKey, keyIdentifier, keyIdentifier, "", systemKeyCode, systemKeyCode, autoRepeat, isKeypad, isSystemKey); err != nil {
		return err
	}
	if _, err := e.tab.Input.DispatchKeyEvent("char", modifiers, timestamp, systemKey, unmodifiedText, "", "", "", 0, 0, autoRepeat, isKeypad, isSystemKey); err != nil {
		return err
	}
	return nil
}

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
