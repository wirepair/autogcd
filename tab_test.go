package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
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

	timeout := time.NewTimer(5 * time.Second)
	go func() {
		for {
			select {
			case <-timeout.C:
				t.Fatalf("timed out waiting for console message")
			}
		}
	}()

	wg.Wait()

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
