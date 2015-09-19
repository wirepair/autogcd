package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
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

	if _, err := tab.Navigate(testServerAddr); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	doc, err := tab.GetElementsBySelector("html")
	if err != nil {
		t.Fatalf("error getting html doc elementL %s\n", err)
	}

	dimensions, err := doc[0].Dimensions()
	if err != nil {
		t.Fatalf("error getting doc dimensions: %s\n", err)
	}

	x, y, err := centroid(dimensions)
	if err != nil {
		t.Fatalf("error getting centroid of doc: %s\n", err)
	}
	t.Logf("x: %d y: %d\n", x, y)
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
			callerTab.StopConsoleMessages()
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

	if src != "<button id=\"button\"></button>" {
		t.Fatalf("expected <button id=\"button\"></button> but got: %s\n", src)
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
			callerTab.StopConsoleMessages()
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

func TestGoogleSearch(t *testing.T) {
	var err error
	var ele *Element
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	_, err = tab.Navigate("https://www.google.com")
	if err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	ele, _, err = tab.GetElementById("lst-ib")
	if err != nil {
		t.Fatalf("error finding input: %s %#v\n", err, ele)
	}
	ele.SendKeys("gcd\n")
	time.Sleep(5 * time.Second)
}
