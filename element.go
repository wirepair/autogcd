package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
)

type InvalidDimensionsErr struct {
	Message string
}

func (e *InvalidDimensionsErr) Error() string {
	return "invalid dimensions " + e.Message
}

type Element struct {
	tab  *Tab            // reference to the containing tab
	Node *gcdapi.DOMNode // the dom node, taken from the document
	Id   int             // nodeId in chrome
}

func newElement(tab *Tab, node *gcdapi.DOMNode) *Element {
	e := &Element{}
	e.tab = tab
	e.Node = node
	e.Id = node.NodeId
	return e
}

// Get attributes of the node in sequential name,value pairs in the slice.
func (e *Element) GetAttributes() (map[string]string, error) {
	attributes := make(map[string]string)
	attr, err := e.tab.DOM.GetAttributes(e.Id)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(attr); i += 2 {
		attributes[attr[i]] = attr[i+1]
	}
	return attributes, nil
}

// Works like WebDriver's clear(), simply sets the attribute value for input.
// or clears the value for textarea.
func (e *Element) Clear() error {
	var err error
	if e.Node.NodeName == "textarea" {
		_, err = e.tab.DOM.SetNodeValue(e.Id, "")
	}
	if e.Node.NodeName == "input" {
		_, err = e.tab.DOM.SetAttributeValue(e.Id, "value", "")
	}
	return err
}

// Returns the outer html of the element.
func (e *Element) GetSource() (string, error) {
	return e.tab.DOM.GetOuterHTML(e.Id)
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
	box, err := e.tab.DOM.GetBoxModel(e.Id)
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
