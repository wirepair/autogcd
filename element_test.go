package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
	"log"
	"sync"
	"testing"
	"time"
)

func TestElementDimensions(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "/button.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	log.Printf("getting buttons")
	buttons, err := tab.GetElementsBySelector("button")
	if err != nil {
		t.Fatalf("error getting buttons %s\n", err)
	}
	log.Printf("got buttons")
	for _, button := range buttons {
		dimensions, err := button.Dimensions()
		if err != nil {
			t.Fatalf("error getting doc dimensions: %s\n", err)
		}

		x, y, err := centroid(dimensions)
		if err != nil {
			t.Fatalf("error getting centroid of doc: %s\n", err)
		}
		t.Logf("x: %d y: %d\n", x, y)
	}

}

func TestElementClick(t *testing.T) {
	var buttons []*Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "button.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	buttons, err = tab.GetElementsBySelector("button")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	if len(buttons) == 0 {
		t.Fatal("no buttons found")
	}

	err = buttons[0].Click()
	if err != nil {
		t.Fatalf("error clicking button: %s\n", err)
	}

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		t.Log("Got message %v\n", message)
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

func TestElementGetSource(t *testing.T) {
	var ele []*Element
	var src string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "button.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	ele, err = tab.GetElementsBySelector("button")
	if err != nil {
		t.Fatalf("error finding buttons: %s\n", err)
	}

	if len(ele) == 0 {
		t.Fatal("no element found")
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

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "attributes.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
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

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		t.Logf("got message: %v\n", message)
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

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "attributes.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	ele, _, err = tab.GetElementById("attr")
	if err != nil {
		t.Fatalf("error finding input: %s %#v\n", err, ele)
	}

	ele.WaitForReady()
	tagName, err := ele.GetTagName()
	if err != nil {
		t.Fatalf("Error getting tagname!")
	}
	t.Logf("ele ready: tagname: " + tagName)
	if tagName != "input" {
		t.Fatalf("Error expected tagname to be input got: %s\n", tagName)
	}
}

func TestElementGetEventListeners(t *testing.T) {
	var err error
	var ele *Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "events.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	ele, _, err = tab.GetElementById("divvie")
	if err != nil {
		t.Fatalf("error finding input: %s %#v\n", err, ele)
	}
	ele.WaitForReady()
	listeners, err := ele.GetEventListeners()
	for _, listener := range listeners {
		t.Logf("%#v\n", listener)
		src, err := tab.GetScriptSource(listener.Location.ScriptId)
		if err != nil {
			t.Fatalf("error getting source: %s\n", err)
		}
		t.Logf("script source: %s\n", src)
	}
}

func TestElementFrameGetTag(t *testing.T) {
	var err error
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "iframe.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	frames := tab.GetFrameDocuments()

	if len(frames) < 2 {
		t.Fatalf("error finding frame element\n")
	}
	t.Logf("got %d frame documents\n", len(frames))
	var nodeId = -1
	for _, fr := range frames {
		url, _ := tab.GetDocumentCurrentUrl(fr.NodeId())
		if url == testServerAddr+"inner.html" {
			nodeId = fr.NodeId()
		}
		t.Logf("frame nodeid %d url: %s\n", fr.NodeId(), url)
	}
	if nodeId == -1 {
		t.Fatalf("error getting inner.html")
	}

	ele, _, err := tab.GetDocumentElementById(nodeId, "output")
	if err != nil {
		t.Fatalf("error finding the div element inside of frame: %s\n", err)
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

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate(testServerAddr + "invalidated.html")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	//tab.ChromeTarget.DebugEvents(true)

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
	time.Sleep(3 * time.Second)
	if !ele.IsInvalid() {
		t.Fatalf("error child is not invalid after it was removed!")
	}
}
