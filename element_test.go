package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
	"sync"
	"testing"
	"time"
)

var (
	testWaitRate    = 20 * time.Millisecond
	testWaitTimeout = 15 * time.Second
)

func TestElementDimensions(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	tab.Debug(true)
	tab.ChromeTarget.Debug(true)
	t.Logf("serv: %s\n", testServerAddr)
	if _, err := tab.Navigate(testServerAddr + "button.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementsBySelectorNotEmpty(tab, "button"))
	if err != nil {
		t.Fatalf("error finding buttons, timed out waiting: %s\n", err)
	}

	buttons, err := tab.GetElementsBySelector("button")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	for _, button := range buttons {
		dimensions, err := button.Dimensions()
		if err != nil {
			t.Fatalf("error getting doc dimensions: %s\n", err)
		}

		_, _, err = centroid(dimensions)
		if err != nil {
			t.Fatalf("error getting centroid of doc: %s\n", err)
		}
		//t.Logf("x: %d y: %d\n", x, y)
	}

}

func TestElementClick(t *testing.T) {
	var buttons []*Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	_, err = tab.Navigate(testServerAddr + "button.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementsBySelectorNotEmpty(tab, "button"))
	if err != nil {
		t.Fatalf("error finding buttons, timed out waiting: %s\n", err)
	}

	buttons, err = tab.GetElementsBySelector("button")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	err = buttons[0].Click()
	if err != nil {
		t.Fatalf("error clicking button: %s\n", err)
	}

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		//t.Logf("Got message %v\n", message)
		if message.Text == "button clicked" {
			callerTab.StopConsoleMessages(true)
			wg.Done()
		}
	}
	tab.GetConsoleMessages(msgHandler)

	timeout := time.NewTimer(time.Second * 8)
	go func() {
		select {
		case <-timeout.C:
			t.Fatalf("timed out waiting for button click event message")
		}
	}()

	wg.Wait()
}

func TestElementMouseOver(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)
	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		t.Logf("Got message %v\n", message)
		if message.Text == "moused over" {
			callerTab.StopConsoleMessages(true)
			wg.Done()
		}
	}
	tab.GetConsoleMessages(msgHandler)

	_, err = tab.Navigate(testServerAddr + "mouseover.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	/*
		err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "button"))
		if err != nil {
			t.Fatalf("error finding buttons, timed out waiting: %s\n", err)
		}*/

	button, _, err := tab.GetElementById("button")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	err = button.MouseOver()
	if err != nil {
		t.Fatalf("error moving mouse over button: %s\n", err)
	}

	timeout := time.NewTimer(time.Second * 8)
	go func() {
		select {
		case <-timeout.C:
			t.Fatalf("timed out waiting for button click event message")
		}
	}()

	wg.Wait()
}

func TestElementDoubleClick(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)
	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		//t.Logf("Got message %v\n", message)
		if message.Text == "double clicked" {
			callerTab.StopConsoleMessages(true)
			wg.Done()
		}
	}
	tab.GetConsoleMessages(msgHandler)

	_, err = tab.Navigate(testServerAddr + "dblclick.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "doubleclick"))
	if err != nil {
		t.Fatalf("error finding buttons, timed out waiting: %s\n", err)
	}

	div, _, err := tab.GetElementById("doubleclick")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	err = div.DoubleClick()
	if err != nil {
		t.Fatalf("error double clicking div: %s\n", err)
	}

	timeout := time.NewTimer(time.Second * 5)
	go func(timeout *time.Timer) {
		select {
		case <-timeout.C:
			t.Fatalf("timed out waiting for dblclick event message")
		}
	}(timeout)

	wg.Wait()
}

func TestElementGetSource(t *testing.T) {
	var ele []*Element
	var src string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	_, err = tab.Navigate(testServerAddr + "button.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementsBySelectorNotEmpty(tab, "button"))
	if err != nil {
		t.Fatalf("error finding buttons, timed out waiting: %s\n", err)
	}

	ele, err = tab.GetElementsBySelector("button")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	src, err = ele[0].GetSource()
	if err != nil {
		t.Fatalf("error getting element source: %s\n", err)
	}

	if src != "<button id=\"button\">click me</button>" {
		t.Fatalf("expected <button id=\"button\">click me</button> but got: %s\n", src)
	}
}

func TestElementGetAttributes(t *testing.T) {
	var err error
	var ele *Element
	var attrs map[string]string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	_, err = tab.Navigate(testServerAddr + "attributes.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "attr"))
	if err != nil {
		t.Fatalf("error finding attr, timed out waiting: %s\n", err)
	}

	ele, _, err = tab.GetElementById("attr")
	if err != nil {
		t.Fatalf("error finding input: %s %#v\n", err, ele)
	}

	attrs, err = ele.GetAttributes()
	if err != nil {
		t.Fatalf("error getting attributes: %s\n", err)
	}

	if attrs["type"] != "text" {
		t.Fatalf("type attribute incorrect")
	}

	if attrs["name"] != "attrtest" {
		t.Fatalf("name attribute incorrect")
	}

	if attrs["id"] != "attr" {
		t.Fatalf("id attribute incorrect")
	}

	if attrs["x"] != "y" {
		t.Fatalf("x attribute incorrect")
	}

	if attrs["z"] != "1" {
		t.Fatalf("z attribute incorrect")
	}

	if attrs["disabled"] != "" {
		t.Fatalf("disabled attribute incorrect")
	}
}

func TestElementSendKeys(t *testing.T) {
	var err error
	var ele *Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		//t.Logf("got message: %v\n", message)
		if message.Text == "zomgs Test!" {
			callerTab.StopConsoleMessages(true)
			wg.Done()
		}

	}
	tab.GetConsoleMessages(msgHandler)

	_, err = tab.Navigate(testServerAddr + "input.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "attr"))
	if err != nil {
		src, _ := tab.GetPageSource(0)
		t.Fatalf("error finding attr, timed out waiting: %s\nsrc: %s\n", err, src)
	}

	ele, _, err = tab.GetElementById("attr")
	if err != nil {
		t.Fatalf("error finding input attr: %s\n", err)
	}

	err = ele.SendKeys("zomgs Test!\n")
	if err != nil {
		t.Fatalf("error sending keys: %s\n", err)
	}
	wg.Wait()
}

func TestElementGetTag(t *testing.T) {
	var err error
	var ele *Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	_, err = tab.Navigate(testServerAddr + "attributes.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "attr"))
	if err != nil {
		t.Fatalf("error finding attr, timed out waiting: %s\n", err)
	}

	ele, _, err = tab.GetElementById("attr")
	if err != nil {
		t.Fatalf("error finding input: %s %#v\n", err, ele)
	}

	tagName, err := ele.GetTagName()
	if err != nil {
		t.Fatalf("Error getting tagname!")
	}
	//t.Logf("ele ready: tagname: " + tagName)
	if tagName != "input" {
		t.Fatalf("Error expected tagname to be input got: %s\n", tagName)
	}
}

func TestElementGetEventListeners(t *testing.T) {
	var err error
	var ele *Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	_, err = tab.Navigate(testServerAddr + "events.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "divvie"))
	if err != nil {
		t.Fatalf("error finding divvie, timed out waiting: %s\n", err)
	}

	ele, _, err = tab.GetElementById("divvie")
	if err != nil {
		t.Fatalf("error finding input: %s %#v\n", err, ele)
	}
	ele.WaitForReady()
	listeners, err := ele.GetEventListeners()
	for _, listener := range listeners {
		//t.Logf("%#v\n", listener)
		_, err := tab.GetScriptSource(listener.Location.ScriptId)
		if err != nil {
			t.Fatalf("error getting source: %s\n", err)
		}
		//t.Logf("script source: %s\n", src)
	}
}

// This test kind of sucks because we can actually get multiple #documents
// back from the debugger with unique nodeIds. I do not see any way in which
// you can tell which one is valid.
func TestElementFrameGetTag(t *testing.T) {
	var err error
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "iframe.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	//tab.Debug(true)

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "innerfr"))
	if err != nil {
		t.Fatalf("error finding innerfr, timed out waiting: %s\n", err)
	}

	ifr, _, err := tab.GetElementById("innerfr")
	if err != nil {
		t.Fatalf("error getting inner frame element")
	}
	ifrDocNodeId, err := ifr.GetFrameDocumentNodeId()
	if err != nil {
		t.Fatalf("error getting inner frame's document node id")
	}

	ele, _, err := tab.GetDocumentElementById(ifrDocNodeId, "output")
	if err != nil {
		t.Fatalf("error finding the div element inside of frame nodeId: %d: %s\n", ifrDocNodeId, err)
	}

	err = ele.WaitForReady()
	if err != nil {
		t.Fatalf("timed out waiting for frame element")
	}

	tagName, err := ele.GetTagName()
	if err != nil || tagName != "div" {
		t.Fatalf("error getting tag name of element inside of frame: %s tag: %s", err, tagName)
	}
}

func TestElementInvalidated(t *testing.T) {
	var err error
	var ele *Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	//tab.Debug(true)

	_, err = tab.Navigate(testServerAddr + "invalidated.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	err = tab.WaitFor(testWaitRate, testWaitTimeout, ElementByIdReady(tab, "child"))
	if err != nil {
		t.Fatalf("error finding child, timed out waiting: %s\n", err)
	}

	ele, ready, err := tab.GetElementById("child")
	if err != nil {
		t.Fatalf("error getting child element: %s\n", err)
	}
	if !ready {
		ele.WaitForReady()
	}
	if ele.IsInvalid() {
		t.Fatalf("error child is already invalid!")
	}
	// wait for timeout to removeChild
	time.Sleep(3 * time.Second)
	if !ele.IsInvalid() {
		t.Fatalf("error child is not invalid after it was removed!")
	}
}
