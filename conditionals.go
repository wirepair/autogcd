/*
The MIT License (MIT)

Copyright (c) 2017 isaac dawson

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
		element, _, _ := tab.GetElementById(elementAttributeId)
		return (element != nil) && (element.IsReady())
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

// Returns true when a selector returns a valid list of elements.
func ElementsBySelectorNotEmpty(tab *Tab, elementSelector string) ConditionalFunc {
	return func(tab *Tab) bool {
		eles, err := tab.GetElementsBySelector(elementSelector)
		if err == nil && len(eles) > 0 {
			return true
		}
		return false
	}
}
