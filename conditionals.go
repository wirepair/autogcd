package autogcd

import (
	"strings"
)

// Returns true when the current url equals the equalsUrl
func UrlEquals(tab *Tab, equalsUrl string) ConditionalFunc {
	return func(tab *Tab) bool {
		if url, err := tab.GetCurrentUrl(); err == nil && url == equalsUrl {
			return true
		}
		return false
	}
}

// Returns true when the current url contains the containsUrl
func UrlContains(tab *Tab, containsUrl string) ConditionalFunc {
	return func(tab *Tab) bool {
		if url, err := tab.GetCurrentUrl(); err == nil && strings.Contains(url, containsUrl) {
			return true
		}
		return false
	}
}

// Returns true when the page title equals the provided equalsTitle
func TitleEquals(tab *Tab, equalsTitle string) ConditionalFunc {
	return func(tab *Tab) bool {
		if pageTitle, err := tab.GetTitle(); err == nil && pageTitle == equalsTitle {
			return true
		}
		return false
	}
}

// Returns true if the searchTitle is contained within the page title.
func TitleContains(tab *Tab, searchTitle string) ConditionalFunc {
	return func(tab *Tab) bool {
		if pageTitle, err := tab.GetTitle(); err == nil && strings.Contains(pageTitle, searchTitle) {
			return true
		}
		return false
	}
}

// Returns true when the element exists and is ready
func ElementByIdReady(tab *Tab, elementAttributeId string) ConditionalFunc {
	return func(tab *Tab) bool {
		_, ready, _ := tab.GetElementById(elementAttributeId)
		return ready
	}
}

// Returns true when the element's attribute of name equals value.
func ElementAttributeEquals(tab *Tab, element *Element, name, value string) ConditionalFunc {
	return func(tab *Tab) bool {
		if element.GetAttribute(name) == value {
			return true
		}
		return false
	}
}
