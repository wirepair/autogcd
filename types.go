package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
)

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

type NetworkRequest struct {
	RequestId        string
	FrameId          string
	LoaderId         string
	DocumentURL      string
	Request          *gcdapi.NetworkRequest
	Timestamp        float64
	Initiator        *gcdapi.NetworkInitiator
	RedirectResponse *gcdapi.NetworkResponse
	Type             string
}

type NetworkResponse struct {
	RequestId   string
	FrameId     string
	LoaderId    string
	DocumentURL string
	Response    *gcdapi.NetworkResponse
	Timestamp   float64
	Type        string
}
