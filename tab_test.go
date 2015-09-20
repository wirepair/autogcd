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
		callerTab.StopConsoleMessages(true)
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
	src, err = tab.GetPageSource("")
	if err != nil {
		t.Fatalf("Error getting page source: %s\n", err)
	}
	t.Logf("source: %s\n", src)
}

func TestTabFrameGetPageSource(t *testing.T) {
	var src string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "iframe.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	ele, _, err := tab.GetElementById("innerfr")
	if err != nil {
		t.Fatalf("error getting inner frame element")
	}
	err = ele.WaitForReady()
	if err != nil {
		t.Fatalf("error waiting for inner frame element")
	}
	if ele.FrameId() == "" {
		t.Fatalf("frameid is empty!")
	}

	src, err = tab.GetPageSource(ele.FrameId())
	if err != nil {
		t.Fatalf("Error getting page source: %s\n", err)
	}
	if !strings.Contains(src, "<div>HELLL") {
		t.Fatalf("error finding dynamically inserted element in source: %s\n", src)
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

func TestTabEvaluateScript(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	wg := &sync.WaitGroup{}
	wg.Add(1) // should be called 2x, one for main page, one for script_inner.html
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

	if _, err := tab.Navigate(testServerAddr + "button.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	res, _, _, errEval := tab.EvaluateScript("JSON.stringify(document)")
	if errEval != nil {
		t.Fatalf("error evaluating script: %s\n", errEval)
	}
	// trigger completion
	_, _, _, errEval = tab.EvaluateScript("console.log('inject ' + location.href);")
	if errEval != nil {
		t.Fatalf("error evaluating trigger script: %s\n", errEval)
	}
	wg.Wait()
	t.Logf("res: %#v\n", res)
}

func TestTabTwoTabCookies(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	tab1, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	tab2, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab1.Navigate(testServerAddr + "cookie1.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	cookies1, err := tab1.GetCookies()
	if err != nil {
		t.Fatalf("Error getting first tab cookies: %s\n", err)
	}
	for _, cookie := range cookies1 {
		t.Logf("%#v\n", cookie)
	}

	if _, err := tab2.Navigate(testServerAddr + "cookie2.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	cookies2, err := tab2.GetCookies()
	if err != nil {
		t.Fatalf("Error getting second tab cookies: %s\n", err)
	}
	for _, cookie := range cookies2 {
		t.Logf("%#v\n", cookie)
	}

	// oddly this returns tab2's cookies :<
	cookies3, err := tab1.GetCookies()
	if err != nil {
		t.Fatalf("Error getting tab1 cookies again: %s\n", err)
	}
	for _, cookie := range cookies3 {
		t.Logf("%#v\n", cookie)
	}
}

func TestTabNetworkListen(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	tab1, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	requestHandlerFn := func(callerTab *Tab, request *NetworkRequest) {
		t.Logf("Got network request: %#v\n", request)
	}
	responseHandlerFn := func(callerTab *Tab, response *NetworkResponse) {
		t.Logf("got a network response: %#v\n", response)
	}
	if err := tab1.ListenNetworkTraffic(requestHandlerFn, responseHandlerFn); err != nil {
		t.Fatalf("Error listening to network traffic: %s\n", err)
	}
	if _, err := tab1.Navigate(testServerAddr + "button.html"); err != nil {
		t.Fatalf("error navigating to target: %s\n", err)
	}
	tab2, err := testAuto.NewTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	if _, err := tab2.Navigate(testServerAddr + "console_log.html"); err != nil {
		t.Fatalf("error navigating to target: %s\n", err)
	}
}

func testTimeout(t *testing.T, duration time.Duration) {
	time.Sleep(duration)
	t.Fatalf("timed out waiting for console message")
}
