package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTabNavigate(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate("http://google.com"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
}

func TestTabGetConsoleMessage(t *testing.T) {
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
		callerTab.StopConsoleMessages()
		wg.Done()
	}
	tab.GetConsoleMessages(msgHandler)

	if _, err := tab.Navigate(testServerAddr + "console_log.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	go testTimeout(t, 5)
	wg.Wait()

}

func TestTabGetDocument(t *testing.T) {
	var doc *gcdapi.DOMNode
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err = tab.Navigate(testServerAddr + "attributes.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	doc, err = tab.GetDocument()
	if err != nil {
		t.Fatalf("error getting doc: %s\n", err)
	}
	testPrintNodes(t, doc)
}

func testPrintNodes(t *testing.T, node *gcdapi.DOMNode) {
	for _, childNode := range node.Children {
		t.Logf("%#v\n\n", childNode)
		testPrintNodes(t, childNode)
	}
}

func TestTabGetPageSource(t *testing.T) {
	var src string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "inner.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	src, err = tab.GetPageSource()
	if err != nil {
		t.Fatalf("Error getting page source: %s\n", err)
	}
	t.Logf("source: %s\n", src)
}

func TestTabGetFrameResources(t *testing.T) {
	var resourceMap map[string]string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "iframe.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	resourceMap, err = tab.GetFrameResources()
	if err != nil {
		t.Fatalf("Error getting page source: %s\n", err)
	}
	for k, v := range resourceMap {
		t.Logf("id: %s url: %s\n", k, v)
		src, wasBase64, err := tab.GetFrameSource(k, v)
		if err != nil {
			t.Fatalf("error getting frame source: %s\n", err)
		}
		t.Logf("wasBase64: %v src: %s\n", wasBase64, src)
	}

}

func TestTabPromptHandler(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	promptHandlerFn := func(theTab *Tab, message, promptType string) {
		if promptType == "prompt" {
			theTab.Page.HandleJavaScriptDialog(true, "someinput")
		}
	}
	tab.SetJavaScriptPromptHandler(promptHandlerFn)
	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		if message.Text == "someinput" {
			wg.Done()
		}
	}
	tab.GetConsoleMessages(msgHandler)

	if _, err := tab.Navigate(testServerAddr + "prompt.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	go testTimeout(t, 5)
	wg.Wait()
}

func TestTabNavigationTimeout(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	tab.SetNavigationTimeout(10)
	if _, err := tab.Navigate(testServerAddr + "prompt.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
}

func TestTabInjectScript(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	wg := &sync.WaitGroup{}
	wg.Add(4) // should be called 2x, one for main page, one for script_inner.html
	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		if strings.Contains(message.Text, "inject") {
			t.Logf("got message: %s\n", message.Text)
			wg.Done()
		}
	}
	tab.GetConsoleMessages(msgHandler)
	tab.InjectScriptOnLoad("console.log('inject ' + location.href);")
	tab.InjectScriptOnLoad("console.log('inject 2' + location.href);")
	if _, err := tab.Navigate(testServerAddr + "script.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	wg.Wait()
}

func testTimeout(t *testing.T, duration time.Duration) {
	time.Sleep(duration)
	t.Fatalf("timed out waiting for console message")
}
