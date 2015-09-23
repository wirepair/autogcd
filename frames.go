package autogcd

import (
	"encoding/base64"
	"encoding/json"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
	"log"
	"sync"
	"time"
)

type Frame struct {
	DOMElement
	documentURL string
	baseURL     string
	mimeType    string
	url         string
	frameName   string

	eleMutex          *sync.RWMutex
	Elements          map[int]*Element
	frameId           string
	parentId          string
	parentFrameNodeId int
}

func newFrame(tab *Tab, frameId, parentId, url, mimeType, frameName string) *Frame {
	f := &Frame{}
	f.frameId = frameId
	f.parentId = parentId
	f.url = url
	f.mimeType = mimeType
	f.frameName = frameName
	f.eleMutex = &sync.RWMutex{}
	f.Elements = make(map[int]*Element)
	f.readyGate = make(chan struct{})
}

func (f *Frame) populateFrame(parentFrameNodeId int, contentDocument *gcdapi.DOMNode, documentURL, baseURL string) {
	f.node = contentDocument
	f.parentId = parentFrameNodeId
	f.documentURL = documentURL
	f.baseURL = baseURL
	// close it
	if !e.ready {
		close(e.readyGate)
	}
	e.ready = true
}

// if no parent id, we are the top Frame.
func (f *Frame) IsTop() bool {
	return f.parentId == ""
}

func (f *Frame) getChildElements() error {
	_, err = t.DOM.RequestChildNodes(f.id, -1)
	return err
}
