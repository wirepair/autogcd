package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
	"time"
)

// a set of properties and functions that are shared amoung elements and frames
type DOMElement struct {
	tab         *Tab            // reference to the containing tab
	node        *gcdapi.DOMNode // the dom node, taken from the document
	readyGate   chan struct{}
	id          int  // nodeId in chrome
	ready       bool // has this elements data been populated by setChildNodes or GetDocument?
	invalidated bool // has this node been invalidated (removed?)
}

// Has the Chrome Debugger notified us of this Elements data yet?
func (d *DOMElement) IsReady() bool {
	return (d.ready && !d.invalidated)
}

// Has the debugger invalidated (removed) the element from the DOM?
func (d *DOMElement) IsInvalid() bool {
	return d.invalidated
}

// The element has become invalid.
func (d *DOMElement) setInvalidated(invalid bool) {
	d.invalidated = invalid
}

// If we are ready, just return, if we are not, wait for the readyGate
// to be closed or for the timeout timer to fird.
func (d *DOMElement) WaitForReady() error {
	if d.ready {
		return nil
	}

	timeout := time.NewTimer(d.tab.elementTimeout * time.Second)
	select {
	case <-d.readyGate:
		return nil
	case <-timeout.C:
		return &ElementNotReadyErr{}
	}
}

// Returns the outer html of the element.
func (d *DOMElement) GetSource() (string, error) {
	if d.invalidated {
		return "", &InvalidElementErr{}
	}
	return d.tab.DOM.GetOuterHTML(d.id)
}

// Returns event listeners for the element, both static and dynamically bound.
func (d *DOMElement) GetEventListeners() ([]*gcdapi.DOMDebuggerEventListener, error) {
	rro, err := d.tab.DOM.ResolveNode(d.id, "")
	if err != nil {
		return nil, err
	}
	eventListeners, err := d.tab.DOMDebugger.GetEventListeners(rro.ObjectId)
	if err != nil {
		return nil, err
	}
	return eventListeners, nil
}
