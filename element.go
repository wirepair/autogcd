package autogcd

import (
	"fmt"
	"github.com/wirepair/gcd/gcdapi"
	"strings"
	"time"
)

// An attempt was made to execute a method for which the element type is incorrect
type IncorrectElementTypeErr struct {
	NodeName     string
	ExpectedName string
}

func (e *IncorrectElementTypeErr) Error() string {
	return "incorrect element type, expected " + e.ExpectedName + " but this type is " + e.NodeName
}

// The element has been removed from the DOM
type InvalidElementErr struct {
}

func (e *InvalidElementErr) Error() string {
	return "this element has been invalidated"
}

// The element has no children
type ElementHasNoChildrenErr struct {
}

func (e *ElementHasNoChildrenErr) Error() string {
	return "this element has no child elements"
}

// for when we have an element that has not been populated
// with data yet.
type ElementNotReadyErr struct {
}

func (e *ElementNotReadyErr) Error() string {
	return "this element is not ready"
}

// for when the dimensions of an element are incorrect to calculate the centroid
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
	attributes    map[string]string // dom attributes
	nodeName      string
	characterData string
	nodeType      int
	tab           *Tab            // reference to the containing tab
	node          *gcdapi.DOMNode // the dom node, taken from the document
	readyGate     chan struct{}
	id            int  // nodeId in chrome
	ready         bool // has this elements data been populated by setChildNodes or GetDocument?
	invalidated   bool // has this node been invalidated (removed?)

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

// populate the Element with node data.
func (e *Element) populateElement(node *gcdapi.DOMNode) {
	e.node = node
	e.nodeType = node.NodeType
	e.nodeName = strings.ToLower(node.NodeName)
	for i := 0; i < len(node.Attributes); i += 2 {
		e.updateAttribute(node.Attributes[i], node.Attributes[i+1])
	}

	// close it
	if !e.ready {
		close(e.readyGate)
	}
	e.ready = true
}

// Has the Chrome Debugger notified us of this Elements data yet?
func (e *Element) IsReady() bool {
	return (e.ready && !e.invalidated)
}

// Has the debugger invalidated (removed) the element from the DOM?
func (e *Element) IsInvalid() bool {
	return e.invalidated
}

// The element has become invalid.
func (e *Element) setInvalidated(invalid bool) {
	e.invalidated = invalid
}

// If we are ready, just return, if we are not, wait for the readyGate
// to be closed or for the timeout timer to fird.
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

// Returns the outer html of the element.
func (e *Element) GetSource() (string, error) {
	if e.invalidated {
		return "", &InvalidElementErr{}
	}
	return e.tab.DOM.GetOuterHTML(e.id)
}

// Is this Element a #document?
func (e *Element) IsDocument() (bool, error) {
	if !e.ready {
		return false, &ElementNotReadyErr{}
	}
	return (e.nodeType == int(DOCUMENT_NODE)), nil
}

// If this is a #document, returns the underlying chrome frameId.
func (e *Element) FrameId() (string, error) {
	isDoc, err := e.IsDocument()
	if err != nil {
		return "", err
	}

	if !isDoc {
		return "", nil
	}

	return e.node.FrameId, nil
}

// If this element is a frame or iframe, return the ContentDocument node id
func (e *Element) GetFrameDocumentNodeId() (int, error) {
	if !e.ready {
		return -1, &ElementNotReadyErr{}
	}
	if e.node.ContentDocument != nil {
		return e.node.ContentDocument.NodeId, nil
	}
	return -1, &IncorrectElementTypeErr{ExpectedName: "(i)frame", NodeName: e.nodeName}
}

// Returns the node id of this Element
func (e *Element) NodeId() int {
	return e.id
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

// Returns the underlying DOMNode for this element.
func (e *Element) GetDebuggerDOMNode() (*gcdapi.DOMNode, error) {
	if !e.ready {
		return nil, &ElementNotReadyErr{}
	}
	if e.invalidated {
		return nil, &InvalidElementErr{}
	}
	return e.node, nil
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
	e.node.ChildNodeCount = newValue
}

func (e *Element) addChild(child *gcdapi.DOMNode) {
	if e.node.Children == nil {
		e.node.Children = make([]*gcdapi.DOMNode, 0)
	}
	e.node.Children = append(e.node.Children, child)
	e.node.ChildNodeCount++
}

func (e *Element) addChildren(childNodes []*gcdapi.DOMNode) {
	for _, child := range childNodes {
		e.addChild(child)
	}
}

func (e *Element) removeChild(removedNode *gcdapi.DOMNode) {
	var idx int
	var child *gcdapi.DOMNode
	childIdx := -1
	if e.node == nil || e.node.Children == nil {
		return
	}
	for idx, child = range e.node.Children {
		if child.NodeId == removedNode.NodeId {
			childIdx = idx
			break
		}
	}
	// remove the child via idx from our slice
	if childIdx == -1 {
		return
	}

	e.node.Children = append(e.node.Children[:childIdx], e.node.Children[:childIdx+1]...)
	e.node.ChildNodeCount = e.node.ChildNodeCount - 1
}

// Get child node ids, returns nil if node is not read
func (e *Element) GetChildNodeIds() ([]int, error) {
	if !e.ready {
		return nil, &ElementNotReadyErr{}
	}
	if e.node == nil || e.node.Children == nil {
		return nil, &ElementHasNoChildrenErr{}
	}

	ids := make([]int, len(e.node.Children))
	for k, child := range e.node.Children {
		ids[k] = child.NodeId
	}
	return ids, nil
}

// Returns the tag name (input, div etc) if the element is in a ready state.
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

// Returns true if the node is enabled, only makes sense for form controls.
// Element must be in a ready state.
func (e *Element) IsEnabled() (bool, error) {
	if !e.ready {
		return false, &ElementNotReadyErr{}
	}
	disabled, ok := e.attributes["disabled"]
	// if the attribute doesn't exist, it's enabled.
	if !ok {
		return true, nil
	}
	if disabled == "true" || disabled == "" {
		return false, nil
	}
	return true, nil
}

// Returns the CSS Style Text of the element, returns the inline style first
// and the attribute style second, or error.
func (e *Element) GetCssInlineStyleText() (string, string, error) {
	inline, attribute, err := e.tab.CSS.GetInlineStylesForNode(e.id)
	if err != nil {
		return "", "", err
	}
	return inline.CssText, attribute.CssText, nil
}

// Returns all of the computed css styles in form of name value map.
func (e *Element) GetComputedCssStyle() (map[string]string, error) {
	styles, err := e.tab.CSS.GetComputedStyleForNode(e.id)
	if err != nil {
		return nil, err
	}
	styleMap := make(map[string]string, len(styles))
	for _, style := range styles {
		styleMap[style.Name] = style.Value
	}
	return styleMap, nil
}

// Get attributes of the node returning a map of name,value pairs.
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

func (e *Element) GetAttribute(name string) string {
	attr, err := e.GetAttributes()
	if err != nil {
		return ""
	}
	return attr[name]
}

// Works like WebDriver's clear(), simply sets the attribute value for input
// or clears the value for textarea. This element must be ready so we can
// properly read the nodeName value.
func (e *Element) Clear() error {
	var err error

	if !e.ready {
		return &ElementNotReadyErr{}
	}

	if e.nodeName != "textarea" || e.nodeName != "input" {
		return &IncorrectElementTypeErr{ExpectedName: "textarea or input", NodeName: e.nodeName}
	}

	if e.nodeName == "textarea" {
		_, err = e.tab.DOM.SetNodeValue(e.id, "")
	}
	if e.nodeName == "input" {
		_, err = e.tab.DOM.SetAttributeValue(e.id, "value", "")
	}
	return err
}

// Clicks the center of the element.
func (e *Element) Click() error {
	x, y, err := e.getCenter()
	if err != nil {
		return err
	}

	// click the centroid of the element.
	return e.tab.Click(x, y)
}

func (e *Element) DoubleClick() error {
	x, y, err := e.getCenter()
	if err != nil {
		return err
	}
	return e.tab.DoubleClick(x, y)
}

func (e *Element) Focus() error {
	_, err := e.tab.DOM.Focus(e.id)
	return err
}

// moves the mouse over the center of the element.
func (e *Element) MouseOver() error {
	x, y, err := e.getCenter()
	if err != nil {
		return err
	}
	return e.tab.MoveMouse(x, y)
}

// gets the center of the element
func (e *Element) getCenter() (int, int, error) {
	points, err := e.Dimensions()
	if err != nil {
		return 0, 0, err
	}

	x, y, err := centroid(points)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

// SendKeys - sends each individual character after focusing (clicking) on the element.
// Extremely basic, doesn't take into account most/all system keys except enter, tab or backspace.
func (e *Element) SendKeys(text string) error {
	e.Focus()
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

// Gnarly output mode activated
func (e *Element) String() string {
	output := fmt.Sprintf("NodeId: %d Invalid: %t Ready: %t", e.id, e.invalidated, e.ready)
	if !e.ready {
		return output
	}
	attrs := ""
	for key, value := range e.attributes {
		attrs = attrs + "\t" + key + "=" + value + "\n"
	}
	output = fmt.Sprintf("%s NodeType: %d TagName: %s characterData: %s childNodeCount: %d attributes (%d): \n%s", output, e.nodeType, e.nodeName, e.characterData, e.node.ChildNodeCount, len(e.attributes), attrs)
	if e.nodeType == int(DOCUMENT_NODE) {
		output = fmt.Sprintf("%s FrameId: %s documentURL: %s\n", output, e.node.FrameId, e.node.DocumentURL)
	}
	//output = fmt.Sprintf("%s %#v", output, e.node)
	return output
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
