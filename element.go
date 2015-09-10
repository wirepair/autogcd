package autogcd

import (
	"github.com/wirepair/gcd/gcdprotogen/types"
)

type InvalidDimensionsErr struct {
	Message string
}

func (e *InvalidDimensionsErr) Error() string {
	return "invalid dimensions " + e.Message
}

type Element struct {
	tab *Tab                   // reference to the containing tab
	id  *types.ChromeDOMNodeId // nodeId in chrome
}

func newElement(tab *Tab, id int) *Element {
	nodeId := types.ChromeDOMNodeId(id)
	e := &Element{}
	e.tab = tab
	e.id = &nodeId
	return e
}

// Get attributes of the node in sequential name,value pairs in the slice.
func (e *Element) GetAttributes() (map[string]string, error) {
	attributes := make(map[string]string)
	attr, err := e.tab.DOM().GetAttributes(e.id)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(attr); i += 2 {
		attributes[attr[i]] = attr[i+1]
	}
	return attributes, nil
}

func (e *Element) GetSource() (string, error) {
	return e.tab.DOM().GetOuterHTML(e.id)
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

func (e *Element) SendKeys(text string) error {
	//type ( enumerated string [ "char" , "keyDown" , "keyUp" , "rawKeyDown" ] )
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
	_, err = e.tab.Input().DispatchKeyEvent(theType, modifiers, timestamp, text, unmodifiedText, keyIdentifier, code, key, windowsVirtualKeyCode, nativeVirtualKeyCode, autoRepeat, isKeypad, isSystemKey)
	return err
}

func (e *Element) Dimensions() ([]float64, error) {
	var points []float64
	box, err := e.tab.DOM().GetBoxModel(e.id)
	if err != nil {
		return nil, err
	}
	points = *box.Content
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
