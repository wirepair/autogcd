package autogcd

import (
	"encoding/json"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
)

func (t *Tab) subscribeSetChildNodes() {
	// new nodes
	t.Subscribe("DOM.setChildNodes", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMSetChildNodesEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: SetChildNodesEvent, Nodes: event.Nodes, ParentNodeId: event.ParentId}
		}
	})
}

func (t *Tab) subscribeAttributeModified() {
	t.Subscribe("DOM.attributeModifiedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMAttributeModifiedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: AttributeModifiedEvent, Name: event.Name, Value: event.Value, NodeId: event.NodeId}
		}
	})
}

func (t *Tab) subscribeAttributeRemoved() {
	t.Subscribe("DOM.attributeRemovedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMAttributeRemovedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: AttributeRemovedEvent, NodeId: event.NodeId, Name: event.Name}
		}
	})
}
func (t *Tab) subscribeCharacterDataModified() {
	t.Subscribe("DOM.characterDataModifiedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMCharacterDataModifiedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: CharacterDataModifiedEvent, NodeId: event.NodeId, CharacterData: event.CharacterData}
		}
	})
}
func (t *Tab) subscribeChildNodeCountUpdated() {
	t.Subscribe("DOM.childNodeCountUpdatedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMChildNodeCountUpdatedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: ChildNodeCountUpdatedEvent, NodeId: event.NodeId, ChildNodeCount: event.ChildNodeCount}
		}
	})
}
func (t *Tab) subscribeChildNodeInserted() {
	t.Subscribe("DOM.childNodeInsertedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMChildNodeInsertedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: ChildNodeInsertedEvent, Node: event.Node, ParentNodeId: event.ParentNodeId, PreviousNodeId: event.PreviousNodeId}
		}
	})
}
func (t *Tab) subscribeChildNodeRemoved() {
	t.Subscribe("DOM.childNodeRemovedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		header := &gcdapi.DOMChildNodeRemovedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event := header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: ChildNodeRemovedEvent, ParentNodeId: event.ParentNodeId, NodeId: event.NodeId}
		}
	})
}

/*
func (t *Tab) subscribeInlineStyleInvalidated() {
	t.Subscribe("DOM.inlineStyleInvalidatedEvent", func(target *gcd.ChromeTarget, payload []byte) {
		event := &gcdapi.DOMInlineStyleInvalidatedEvent{}
		err := json.Unmarshal(payload, header)
		if err == nil {
			event = header.Params
			t.nodeChange <- &NodeChangeEvent{EventType: InlineStyleInvalidatedEvent, NodeIds: event.NodeIds}
		}
	})
}
*/
func (t *Tab) subscribeDocumentUpdated() {
	// node ids are no longer valid
	t.Subscribe("DOM.documentUpdated", func(target *gcd.ChromeTarget, payload []byte) {
		t.nodeChange <- &NodeChangeEvent{EventType: DocumentUpdatedEvent}
	})
}
