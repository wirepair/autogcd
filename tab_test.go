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

	if _, err := tab.Navigate(testServerAddr + "index.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
}

func TestTabGetCurrentUrl(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "console.html?x=1"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	url, err := tab.GetCurrentUrl()
	if err != nil {
		t.Fatalf("error getting url: %s\n", err)
	}
	if url != testServerAddr+"console.html?x=1" {
		t.Fatalf("expected url is different, got: %s\n", url)
	}
}

func TestTabGetTitle(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "index.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	title, err := tab.GetTitle()
	if err != nil {
		t.Fatalf("error getting url: %s\n", err)
	}
	if title != "autogcd test" {
		t.Fatalf("expected title is different, got: %s\n", title)
	}
}

func TestTabGetConsoleMessage(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	timeout := time.NewTimer(5 * time.Second)
	done := make(chan struct{})
	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
		if message.Text == "this is a console message" {
			done <- struct{}{}
		}

	}
	tab.GetConsoleMessages(msgHandler)

	if _, err := tab.Navigate(testServerAddr + "console_log.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	select {
	case <-done:
		return
	case <-timeout.C:
		t.Fatalf("error waiting for console message")
	}

}

func TestTabGetPageSource(t *testing.T) {
	//var src string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate(testServerAddr + "inner.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	_, err = tab.GetPageSource(0)
	if err != nil {
		t.Fatalf("Error getting page source: %s\n", err)
	}
	//t.Logf("source: %s\n", src)
}

func TestTabFrameGetPageSource(t *testing.T) {
	var src string
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	tab.Debug(true)
	if _, err := tab.Navigate(testServerAddr + "iframe.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	ele, _, err := tab.GetElementById("innerfr")
	if err != nil {
		t.Fatalf("error getting inner frame element")
	}
	id, err := ele.GetFrameDocumentNodeId()
	if err != nil {
		t.Fatalf("error getting iframe document node id: %s\n", err)
	}

	frameEle, _ := tab.GetElementByNodeId(id)

	src, err = tab.GetPageSource(frameEle.NodeId())
	if err != nil {
		t.Fatalf("Error getting page source: %s\n", err)
	}
	if !strings.Contains(src, "<div>HELLL") {
		t.Fatalf("error finding dynamically inserted element in source: %s\n", src)
	}
	//t.Logf("source: %s\n", src)
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
		//t.Logf("id: %s url: %s\n", k, v)
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

	timeout := time.NewTimer(5 * time.Second)
	done := make(chan struct{})

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
			done <- struct{}{}
		}
	}
	tab.GetConsoleMessages(msgHandler)

	if _, err := tab.Navigate(testServerAddr + "prompt.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	select {
	case <-done:
		return
	case <-timeout.C:
		t.Fatalf("error waiting for console message")
	}
}

// prompts will block navigation from returning
func TestTabNavigationTimeout(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	tab.SetNavigationTimeout(10)
	if _, err := tab.Navigate(testServerAddr + "prompt.html"); err == nil {
		t.Fatalf("did not get an error navigating: %s\n", err)
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
			//t.Logf("got message: %s\n", message.Text)
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
			//t.Logf("got message: %s\n", message.Text)
			wg.Done()
		}
	}
	tab.GetConsoleMessages(msgHandler)

	if _, err := tab.Navigate(testServerAddr + "button.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	_, errEval := tab.EvaluateScript("JSON.stringify(document)")
	if errEval != nil {
		t.Fatalf("error evaluating script: %s\n", errEval)
	}
	// trigger completion
	_, errEval = tab.EvaluateScript("console.log('inject ' + location.href);")
	if errEval != nil {
		t.Fatalf("error evaluating trigger script: %s\n", errEval)
	}
	wg.Wait()
	//t.Logf("res: %#v\n", res)
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

	_, err = tab1.GetCookies()
	if err != nil {
		t.Fatalf("Error getting first tab cookies: %s\n", err)
	}

	if _, err := tab2.Navigate(testServerAddr + "cookie2.html"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}
	_, err = tab2.GetCookies()
	if err != nil {
		t.Fatalf("Error getting second tab cookies: %s\n", err)
	}

	_, err = tab1.GetCookies()
	if err != nil {
		t.Fatalf("Error getting tab1 cookies again: %s\n", err)
	}

}

func TestTabNetworkTraffic(t *testing.T) {
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
	if err := tab1.GetNetworkTraffic(requestHandlerFn, responseHandlerFn); err != nil {
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

func TestTabWindows(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	if _, err := tab.Navigate(testServerAddr + "window_main.html"); err != nil {
		t.Fatalf("error opening first window")
	}
	t.Logf("# of elements: %d\n", len(tab.elements))
	ele, _, err := tab.GetElementById("mainwindow")
	if err != nil {
		t.Fatalf("error getting mainwindow element")
	}
	ele.WaitForReady()
	tabs := testAuto.GetAllTabs()
	if err != nil {
		t.Fatalf("error getting tabs")
	}

	newTabs, err := testAuto.RefreshTabList()
	if err != nil {
		t.Fatalf("Error getting new tabs")
	}

	t.Logf("%d unknown tabs found!", len(newTabs)-len(tabs))

}

func TestTabAfterRedirect(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	tab, err := testAuto.GetTab()
	//tab.ChromeTarget.DebugEvents(true)
	if err != nil {
		t.Fatalf("error getting tab")
	}
	if _, err := tab.Navigate(testServerAddr + "redirect.html"); err != nil {
		t.Fatalf("error opening first window")
	}

	url, err := tab.GetCurrentUrl()
	if err != nil {
		t.Fatalf("error getting url: %s\n", err)
	}

	if url != testServerAddr+"redirect.html" {
		t.Fatalf("url does not match redirect.html got: %s\n", url)
	}

	oldDoc, err := tab.GetDocument()
	if err != nil {
		t.Fatalf("error getting document")
	}

	ele, _, err := tab.GetElementById("child")
	if err != nil {
		t.Fatalf("error getting child element: %s\n", err)
	}

	err = ele.WaitForReady()
	if err != nil {
		t.Fatalf("timed out waiting for element")
	}

	if ele.IsInvalid() {
		t.Fatalf("element is invalid before redirect occurred")
	}

	time.Sleep(4 * time.Second)
	newUrl, err := tab.GetCurrentUrl()
	if err != nil {
		t.Fatalf("error getting url after redirect: %s\n", err)
	}
	if newUrl != testServerAddr+"redirect_target.html" {
		t.Fatalf("url does not match redirect_target.html got: %s\n", newUrl)
	}

	if !ele.IsInvalid() {
		t.Fatalf("error element was not invalidated after redirect!")
	}

	if !oldDoc.IsInvalid() {
		t.Fatalf("error document was not invalidated after redirect!")
	}

}

func TestTabFrameRedirect(t *testing.T) {
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}
	if _, err := tab.Navigate(testServerAddr + "frame_top.html"); err != nil {
		t.Fatalf("error opening first window")
	}

	ifr, ready, err := tab.GetElementById("frameredirect")
	if err != nil {
		t.Fatalf("error finding frame element")
	}

	if !ready {
		ifr.WaitForReady()
	}

	if ifr.IsInvalid() {
		t.Fatalf("error frame was invalidated before redirect")
	}

	// get reference to the frames document before redirect
	ifrDocNodeId, err := ifr.GetFrameDocumentNodeId()
	if err != nil {
		t.Fatalf("error getting doc node of invalidated iframe")
	}
	ifrDoc, _ := tab.GetElementByNodeId(ifrDocNodeId)

	time.Sleep(4 * time.Second)

	if !ifr.IsInvalid() {
		t.Fatalf("error iframe was not invalidated after redirect")
	}

	if !ifrDoc.IsInvalid() {
		t.Fatalf("error the iframe elements document was not invalidated after redirect")
	}
}

func TestTabMultiTab(t *testing.T) {
	numTabs := 5
	tabs := make([]*Tab, numTabs)
	testAuto := testDefaultStartup(t)
	defer testAuto.Shutdown()
	for i := 0; i < numTabs; i++ {
		tab, err := testAuto.NewTab()
		if err != nil {
			t.Fatalf("error opening tab %d %s\n", i, err)
		}
		tabs[i] = tab
	}
	wg := &sync.WaitGroup{}
	for _, tab := range tabs {
		wg.Add(1)
		testMultiNavigateSendKeys(t, wg, tab)
	}
	wg.Wait()
}

func testMultiNavigateSendKeys(t *testing.T, wg *sync.WaitGroup, tab *Tab) {
	var err error
	var ele *Element
	// sleep for a random ms so we aren't attempting to send all events at the same time

	msgHandler := func(callerTab *Tab, message *gcdapi.ConsoleConsoleMessage) {
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
}
