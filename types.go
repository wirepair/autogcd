/*
The MIT License (MIT)

Copyright (c) 2015 isaac dawson

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
	"github.com/wirepair/gcd/gcdapi"
)

// Common node types
type NodeType uint8

const (
	ELEMENT_NODE                NodeType = 0x1
	TEXT_NODE                   NodeType = 0x3
	PROCESSING_INSTRUCTION_NODE NodeType = 0x7
	COMMENT_NODE                NodeType = 0x8
	DOCUMENT_NODE               NodeType = 0x9
	DOCUMENT_TYPE_NODE          NodeType = 0x10
	DOCUMENT_FRAGMENT_NODE      NodeType = 0x11
)

var nodeTypeMap = map[NodeType]string{
	ELEMENT_NODE:                "ELEMENT_NODE",
	TEXT_NODE:                   "TEXT_NODE",
	PROCESSING_INSTRUCTION_NODE: "PROCESSING_INSTRUCTION_NODE",
	COMMENT_NODE:                "COMMENT_NODE",
	DOCUMENT_NODE:               "DOCUMENT_NODE",
	DOCUMENT_TYPE_NODE:          "DOCUMENT_TYPE_NODE",
	DOCUMENT_FRAGMENT_NODE:      "DOCUMENT_FRAGMENT_NODE",
}

// Document/Node change event types
type ChangeEventType uint16

const (
	DocumentUpdatedEvent        ChangeEventType = 0x0
	SetChildNodesEvent          ChangeEventType = 0x1
	AttributeModifiedEvent      ChangeEventType = 0x2
	AttributeRemovedEvent       ChangeEventType = 0x3
	InlineStyleInvalidatedEvent ChangeEventType = 0x4
	CharacterDataModifiedEvent  ChangeEventType = 0x5
	ChildNodeCountUpdatedEvent  ChangeEventType = 0x6
	ChildNodeInsertedEvent      ChangeEventType = 0x7
	ChildNodeRemovedEvent       ChangeEventType = 0x8
)

var changeEventMap = map[ChangeEventType]string{
	DocumentUpdatedEvent:        "DocumentUpdatedEvent",
	SetChildNodesEvent:          "SetChildNodesEvent",
	AttributeModifiedEvent:      "AttributeModifiedEvent",
	AttributeRemovedEvent:       "AttributeRemovedEvent",
	InlineStyleInvalidatedEvent: "InlineStyleInvalidatedEvent",
	CharacterDataModifiedEvent:  "CharacterDataModifiedEvent",
	ChildNodeCountUpdatedEvent:  "ChildNodeCountUpdatedEvent",
	ChildNodeInsertedEvent:      "ChildNodeInsertedEvent",
	ChildNodeRemovedEvent:       "ChildNodeRemovedEvent",
}

func (evt ChangeEventType) String() string {
	if s, ok := changeEventMap[evt]; ok {
		return s
	}
	return ""
}

// For handling DOM updating nodes
type NodeChangeEvent struct {
	EventType      ChangeEventType   // the type of node change event
	NodeId         int               // nodeid of change
	NodeIds        []int             // nodeid of changes for inlinestyleinvalidated
	ChildNodeCount int               // updated childnodecount event
	Nodes          []*gcdapi.DOMNode // Child nodes array. for setChildNodesEvent
	Node           *gcdapi.DOMNode   // node for child node inserted event
	Name           string            // attribute name
	Value          string            // attribute value
	CharacterData  string            // new text value for characterDataModified events
	ParentNodeId   int               // node id for setChildNodesEvent, childNodeInsertedEvent and childNodeRemovedEvent
	PreviousNodeId int               // previous node id for childNodeInsertedEvent

}

// Outbound network requests
type NetworkRequest struct {
	RequestId        string                   // Internal chrome request id
	FrameId          string                   // frame that the request went out on
	LoaderId         string                   // internal chrome loader id
	DocumentURL      string                   // url of the frame
	Request          *gcdapi.NetworkRequest   // underlying Request object
	Timestamp        float64                  // time the request was dispatched
	Initiator        *gcdapi.NetworkInitiator // who initiated the request
	RedirectResponse *gcdapi.NetworkResponse  // non-nil if it was a redirect
	Type             string                   // Document, Stylesheet, Image, Media, Font, Script, TextTrack, XHR, Fetch, EventSource, WebSocket, Other
}

// Inbound network responses
type NetworkResponse struct {
	RequestId string                  // Internal chrome request id
	FrameId   string                  // frame that the request went out on
	LoaderId  string                  // internal chrome loader id
	Response  *gcdapi.NetworkResponse // underlying Response object
	Timestamp float64                 // time the request was received
	Type      string                  // Document, Stylesheet, Image, Media, Font, Script, TextTrack, XHR, Fetch, EventSource, WebSocket, Other
}

// For storage related events.
type StorageEventType uint16

type StorageEvent struct {
	IsLocalStorage bool   // if true, local storage, false session storage
	SecurityOrigin string // origin that this event occurred on
	Key            string // storage key
	NewValue       string // new storage value
	OldValue       string // old storage value
}
