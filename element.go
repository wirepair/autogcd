/*
The MIT License (MIT)

Copyright (c) 2016 isaac dawson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package autogcd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wirepair/gcd/gcdapi"
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

// When we have an element that has not been populated
// with data yet.
type ElementNotReadyErr struct {
}

func (e *ElementNotReadyErr) Error() string {
	return "this element is not ready"
}

// When the dimensions of an element are incorrect to calculate the centroid
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
	lock           *sync.RWMutex     // for protecting read/write access to this Element
	attributes     map[string]string // dom attributes
	nodeName       string            // the DOM tag name
	characterData  string            // the character data (if any, #text only)
	childNodeCount int               // the number of children this element has
	nodeType       int               // the DOM nodeType
	tab            *Tab              // reference to the containing tab
	node           *gcdapi.DOMNode   // the dom node, taken from the document
	readyGate      chan struct{}     // gate to close upon recieving all information from the debugger service
	id             int               // nodeId in chrome
	ready          bool              // has this elements data been populated by setChildNodes or GetDocument?
	invalidated    bool              // has this node been invalidated (removed?)
}

func newElement(tab *Tab, nodeId int) *Element {
	e := &Element{}
	e.tab = tab
	e.attributes = make(map[string]string)
	e.readyGate = make(chan struct{})
	e.lock = &sync.RWMutex{}
	e.id = nodeId
	return e
}

func newReadyElement(tab *Tab, node *gcdapi.DOMNode) *Element {
	e := &Element{}
	e.tab = tab
	e.attributes = make(map[string]string)
	e.readyGate = make(chan struct{})
	e.nodeName = strings.ToLower(node.NodeName)
	e.lock = &sync.RWMutex{}
	e.populateElement(node)
	return e
}

// populate the Element with node data.
func (e *Element) populateElement(node *gcdapi.DOMNode) {
	e.lock.Lock()
	e.node = node
	e.id = node.NodeId
	e.nodeType = node.NodeType
	e.nodeName = strings.ToLower(node.NodeName)
	e.childNodeCount = node.ChildNodeCount
	if node.NodeType == int(TEXT_NODE) {
		e.characterData = node.NodeValue
	}

	e.lock.Unlock()

	for i := 0; i < len(node.Attributes); i += 2 {
		e.updateAttribute(node.Attributes[i], node.Attributes[i+1])
	}

	// close it
	if !e.ready {
		close(e.readyGate)
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.ready = true
}

// Has the Chrome Debugger notified us of this Elements data yet?
func (e *Element) IsReady() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return (e.ready && !e.invalidated)
}

// Has Chrome notified us, but the element is invalid?
func (e *Element) IsReadyInvalid() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return (e.ready && e.invalidated)
}

// Has the debugger invalidated (removed) the element from the DOM?
func (e *Element) IsInvalid() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.invalidated
}

// The element has become invalid.
func (e *Element) setInvalidated(invalid bool) {
	e.lock.Lock()
	e.invalidated = invalid
	e.lock.Unlock()
}

// If we are ready, just return, if we are not, wait for the readyGate
// to be closed or for the timeout timer to fired.
func (e *Element) WaitForReady() error {
	e.lock.RLock()
	ready := e.ready
	e.lock.RUnlock()

	if ready {
		return nil
	}

	timeout := time.NewTimer(e.tab.elementTimeout)
	defer timeout.Stop()

	select {
	case <-e.readyGate:
		return nil
	case <-timeout.C:
		return &ElementNotReadyErr{}
	}
}

// Returns the outer html of the element.
func (e *Element) GetSource() (string, error) {
	e.lock.RLock()
	id := e.id
	e.lock.RUnlock()

	if e.invalidated {
		return "", &InvalidElementErr{}
	}
	e.tab.debugf("id: %d\n", id)
	return e.tab.DOM.GetOuterHTML(id)
}

// Is this Element a #document?
func (e *Element) IsDocument() (bool, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if !e.ready {
		return false, &ElementNotReadyErr{}
	}
	return (e.nodeType == int(DOCUMENT_NODE)), nil
}

// If this is a #document, returns the underlying chrome frameId.
func (e *Element) FrameId() (string, error) {
	if !e.IsReady() {
		return "", &ElementNotReadyErr{}
	}

	isDoc, err := e.IsDocument()
	if err != nil {
		return "", err
	}

	if !isDoc {
		return "", nil
	}
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.node.FrameId, nil
}

// If this element is a frame or iframe, return the ContentDocument node id
func (e *Element) GetFrameDocumentNodeId() (int, error) {
	if !e.IsReady() {
		return -1, &ElementNotReadyErr{}
	}
	e.lock.RLock()
	defer e.lock.RUnlock()

	if e.node != nil && e.node.ContentDocument != nil {
		return e.node.ContentDocument.NodeId, nil
	}
	return -1, &IncorrectElementTypeErr{ExpectedName: "(i)frame", NodeName: e.nodeName}
}

// Returns the underlying chrome debugger node id of this Element
func (e *Element) NodeId() int {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return e.id
}

// Returns event listeners for the element, both static and dynamically bound.
func (e *Element) GetEventListeners() ([]*gcdapi.DOMDebuggerEventListener, error) {
	e.lock.RLock()
	id := e.id
	e.lock.RUnlock()

	params := &gcdapi.DOMResolveNodeParams{
		NodeId: id,
	}

	rro, err := e.tab.DOM.ResolveNodeWithParams(params)
	if err != nil {
		return nil, err
	}
	eventListeners, err := e.tab.DOMDebugger.GetEventListeners(rro.ObjectId, 1, false)
	if err != nil {
		return nil, err
	}
	return eventListeners, nil
}

// Returns the underlying DOMNode for this element. Note this is potentially
// unsafe to access as we give up the ability to lock.
func (e *Element) GetDebuggerDOMNode() (*gcdapi.DOMNode, error) {
	if !e.IsReady() {
		return nil, &ElementNotReadyErr{}
	}

	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.invalidated {
		return nil, &InvalidElementErr{}
	}
	return e.node, nil
}

// updates the attribute name/value pair
func (e *Element) updateAttribute(name, value string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.attributes[name] = value
}

// removes the attribute from our attributes list.
func (e *Element) removeAttribute(name string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.attributes, name)
}

// updates character data
func (e *Element) updateCharacterData(newValue string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.characterData = newValue
}

// updates child node counts.
func (e *Element) updateChildNodeCount(newValue int) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.childNodeCount = newValue
}

// adds the child to our DOMNode.
func (e *Element) addChild(child *gcdapi.DOMNode) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.node == nil {
		return
	}

	if e.node.Children == nil {
		e.node.Children = make([]*gcdapi.DOMNode, 0)
	}
	e.node.Children = append(e.node.Children, child)
	e.childNodeCount++
}

// adds the children to our DOMNode
func (e *Element) addChildren(childNodes []*gcdapi.DOMNode) {
	for _, child := range childNodes {
		if child != nil {
			e.addChild(child)
		}
	}
}

// removes the child from our DOMNode
func (e *Element) removeChild(removedNodeId int) {
	var idx int
	var child *gcdapi.DOMNode

	e.lock.Lock()
	defer e.lock.Unlock()

	if e.node == nil || e.node.Children == nil {
		return
	}

	for idx, child = range e.node.Children {
		if child != nil && child.NodeId == removedNodeId {
			e.node.Children = append(e.node.Children[:idx], e.node.Children[idx+1:]...)
			e.childNodeCount = e.childNodeCount - 1
			break
		}
	}
}

// Get child node ids, returns nil if node is not ready
func (e *Element) GetChildNodeIds() ([]int, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

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
	e.lock.RLock()
	defer e.lock.RUnlock()

	if !e.ready {
		return "", &ElementNotReadyErr{}
	}
	return e.nodeName, nil
}

// Returns the node type if the element is in a ready state.
func (e *Element) GetNodeType() (int, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if !e.ready {
		return -1, &ElementNotReadyErr{}
	}
	return e.nodeType, nil
}

// Returns the character data if the element is in a ready state.
func (e *Element) GetCharacterData() (string, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if !e.ready {
		return "", &ElementNotReadyErr{}
	}

	return e.characterData, nil
}

// Returns true if the node is enabled, only makes sense for form controls.
// Element must be in a ready state.
func (e *Element) IsEnabled() (bool, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

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

// Simulate WebDrivers checked propertyname check
func (e *Element) IsSelected() (bool, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if !e.ready {
		return false, &ElementNotReadyErr{}
	}

	checked, ok := e.attributes["checked"]
	if ok == true && checked != "false" {
		return true, nil
	}
	return false, nil
}

// Returns the CSS Style Text of the element, returns the inline style first
// and the attribute style second, or error.
func (e *Element) GetCssInlineStyleText() (string, string, error) {
	e.lock.RLock()
	inline, attribute, err := e.tab.CSS.GetInlineStylesForNode(e.id)
	e.lock.RUnlock()

	if err != nil {
		return "", "", err
	}
	return inline.CssText, attribute.CssText, nil
}

// Returns all of the computed css styles in form of name value map.
func (e *Element) GetComputedCssStyle() (map[string]string, error) {
	e.lock.RLock()
	styles, err := e.tab.CSS.GetComputedStyleForNode(e.id)
	e.lock.RUnlock()

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
	e.lock.RLock()
	attr, err := e.tab.DOM.GetAttributes(e.id)
	e.lock.RUnlock()

	if err != nil {
		return nil, err
	}
	for i := 0; i < len(attr); i += 2 {
		e.updateAttribute(attr[i], attr[i+1])
	}

	return e.attributes, nil
}

// Gets a single attribute by name, returns empty string if it does not exist
// or is empty.
func (e *Element) GetAttribute(name string) string {
	attr, err := e.GetAttributes()
	if err != nil {
		return ""
	}
	return attr[name]
}

// Similar to above, but works for boolean properties (checked, async etc)
// Returns true if the attribute is set in our known list of attributes
// for this element.
func (e *Element) HasAttribute(name string) bool {
	attr, err := e.GetAttributes()
	if err != nil {
		return false
	}
	_, exists := attr[name]
	return exists
}

// SetAttributeValue sets an element's attribute with name to value.
func (e *Element) SetAttributeValue(name, value string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	_, err := e.tab.DOM.SetAttributeValue(e.id, name, value)
	if err != nil {
		return err
	}

	e.attributes[name] = value

	return nil
}

// Works like WebDriver's clear(), simply sets the attribute value for input
// or clears the value for textarea. This element must be ready so we can
// properly read the nodeName value.
func (e *Element) Clear() error {
	e.lock.RLock()
	defer e.lock.RUnlock()

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
	return e.tab.Click(float64(x), float64(y))
}

// Double clicks the center of the element.
func (e *Element) DoubleClick() error {
	x, y, err := e.getCenter()
	if err != nil {
		return err
	}

	return e.tab.DoubleClick(float64(x), float64(y))
}

// Focus on the element.
func (e *Element) Focus() error {
	e.lock.RLock()
	defer e.lock.RUnlock()

	params := &gcdapi.DOMFocusParams{
		NodeId: e.id,
	}
	_, err := e.tab.DOM.FocusWithParams(params)
	return err
}

// moves the mouse over the center of the element.
func (e *Element) MouseOver() error {
	x, y, err := e.getCenter()
	if err != nil {
		return err
	}
	return e.tab.MoveMouse(float64(x), float64(y))
}

// Returns the dimensions of the element.
func (e *Element) Dimensions() ([]float64, error) {
	var points []float64
	e.lock.RLock()

	params := &gcdapi.DOMGetBoxModelParams{
		NodeId: e.id,
	}
	box, err := e.tab.DOM.GetBoxModelWithParams(params)

	e.lock.RUnlock()

	if err != nil {
		return nil, err
	}
	points = box.Content
	return points, nil
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

// Gnarly output mode activated
func (e *Element) String() string {
	e.lock.RLock()
	defer e.lock.RUnlock()
	output := fmt.Sprintf("NodeId: %d Invalid: %t Ready: %t", e.id, e.invalidated, e.ready)
	if !e.ready {
		return output
	}
	attrs := ""
	for key, value := range e.attributes {
		attrs = attrs + "\t" + key + "=" + value + "\n"
	}
	output = fmt.Sprintf("%s NodeType: %d TagName: %s characterData: %s childNodeCount: %d attributes (%d): \n%s", output, e.nodeType, e.nodeName, e.characterData, e.childNodeCount, len(e.attributes), attrs)
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
