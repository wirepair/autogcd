package autogcd

import (
	//"github.com/wirepair/gcd/gcdprotogen/types"
	"testing"
	"time"
)

func TestTabNavigate(t *testing.T) {
	testAuto := testTabStart(t)
	defer testAuto.debugger.ExitProcess()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate("http://google.com"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	testAuto.debugger.ExitProcess()
}

func TestTabGetConsoleMessage(t *testing.T) {
	testAuto := testTabStart(t)
	defer testAuto.debugger.ExitProcess()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	messageCh := tab.GetConsoleMessages()
	if _, err := tab.Navigate("http://veracode.com"); err != nil {
		t.Fatalf("Error navigating: %s\n", err)
	}

	timeout := time.NewTimer(5 * time.Second)
	for {
		select {
		case consoleMsg, ok := <-messageCh:
			tab.StopConsoleMessages(messageCh)
			t.Logf("got console message: %v %v\n", consoleMsg, ok)
			goto DONE
		case <-timeout.C:
			t.Fatalf("timed out waiting for console message")
		}
	}
DONE:
}

func TestTabGetPageSource(t *testing.T) {
	var src string
	testAuto := testTabStart(t)
	defer testAuto.debugger.ExitProcess()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate("http://localhost:86/inner.html"); err != nil {
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
	testAuto := testTabStart(t)
	defer testAuto.debugger.ExitProcess()

	tab, err := testAuto.GetTab()
	if err != nil {
		t.Fatalf("error getting tab")
	}

	if _, err := tab.Navigate("http://localhost:86/iframe.html"); err != nil {
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

func testTabStart(t *testing.T) *AutoGcd {
	s := NewSettings(testPath, testDir, testPort)
	testAuto := NewAutoGcd(s)
	if err := testAuto.Start(); err != nil {
		t.Fatalf("failed to start chrome: %s\n", err)
	}
	return testAuto
}
