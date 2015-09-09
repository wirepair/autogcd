package autogcd

import (
	"flag"
	"os"
	"testing"
)

var (
	testPath string
	testDir  string
	testPort string
)

func init() {
	flag.StringVar(&testPath, "chrome", "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", "path to Xvfb")
	flag.StringVar(&testDir, "dir", "C:\\temp\\", "user directory")
	flag.StringVar(&testPort, "port", "9222", "Debugger port")
}

func TestMain(m *testing.M) {
	flag.Parse()
	ret := m.Run()
	testCleanUp()
	os.Exit(ret)
}

func testCleanUp() {

}

func TestStart(t *testing.T) {
	s := NewSettings(testPath, testDir, testPort)
	s.SetStartTimeout(15)
	s.SetChromeHost("localhost")
	auto := NewAutoGcd(s)
	if err := auto.Start(); err != nil {
		t.Fatalf("failed to start chrome: %s\n", err)
	}
	auto.debugger.ExitProcess()
}

func TestGetTab(t *testing.T) {
	var err error
	var tab *Tab
	auto := testDefaultStartup(t)
	tab, err = auto.GetTab()
	if err != nil {
		t.Fatalf("Error getting tab: %s\n", err)
	}

	if tab.Target.Type != "page" {
		t.Fatalf("Got tab but wasn't of type Page")
	}
	auto.debugger.ExitProcess()
}

func TestNewTab(t *testing.T) {
	var err error
	//var newTab *Tab
	auto := testDefaultStartup(t)
	tabLen := len(auto.tabs)
	_, err = auto.NewTab()
	if err != nil {
		t.Fatalf("error creating new tab: %s\n", err)
	}

	if tabLen+1 != len(auto.tabs) {
		t.Fatalf("error created new tab but not reflected in our map")
	}

	auto.debugger.ExitProcess()
}

func TestCloseTab(t *testing.T) {
	var err error
	var newTab *Tab
	auto := testDefaultStartup(t)
	tabLen := len(auto.tabs)

	newTab, err = auto.NewTab()
	if err != nil {
		t.Fatalf("error creating new tab: %s\n", err)
	}

	if tabLen+1 != len(auto.tabs) {
		t.Fatalf("error created new tab but not reflected in our map")
	}

	err = auto.CloseTab(newTab)
	if err != nil {
		t.Fatalf("error closing tab")
	}

	if _, err := auto.tabById(newTab.Target.Id); err == nil {
		t.Fatalf("error closed tab still in our map")
	}

	auto.debugger.ExitProcess()
}

func testDefaultStartup(t *testing.T) *AutoGcd {
	s := NewSettings(testPath, testDir, testPort)
	auto := NewAutoGcd(s)
	if err := auto.Start(); err != nil {
		t.Fatalf("failed to start chrome: %s\n", err)
	}
	return auto
}
