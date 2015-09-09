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
	return x / pointLen, y / pointLen, nil
}
