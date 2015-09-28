package autogcd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
)

var (
	testListener   net.Listener
	testPath       string
	testDir        string
	testPort       string
	testServerAddr string
)

var testStartupFlags = []string{"--disable-new-tab-first-run", "--no-first-run", "--disable-popup-blocking"}

func init() {
	flag.StringVar(&testPath, "chrome", "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", "path to chrome/chromium")
	flag.StringVar(&testDir, "dir", "C:\\temp\\", "user directory")
	flag.StringVar(&testPort, "port", "9222", "Debugger port")
}

func TestMain(m *testing.M) {
	flag.Parse()
	testServer()
	ret := m.Run()
	testCleanUp()
	os.Exit(ret)
}

func testCleanUp() {
	testListener.Close()
}

func TestStart(t *testing.T) {
	s := NewSettings(testPath, testRandomDir(t))
	s.SetStartTimeout(15)
	s.SetChromeHost("localhost")
	auto := NewAutoGcd(s)
	defer auto.Shutdown()

	if err := auto.Start(); err != nil {
		t.Fatalf("failed to start chrome: %s\n", err)
	}

}

func TestGetTab(t *testing.T) {
	var err error
	var tab *Tab
	auto := testDefaultStartup(t)
	defer auto.Shutdown()

	tab, err = auto.GetTab()
	if err != nil {
		t.Fatalf("Error getting tab: %s\n", err)
	}

	if tab.Target.Type != "page" {
		t.Fatalf("Got tab but wasn't of type Page")
	}

}

func TestNewTab(t *testing.T) {
	//var newTab *Tab
	auto := testDefaultStartup(t)
	defer auto.Shutdown()

	tabLen := len(auto.tabs)
	_, err := auto.NewTab()
	if err != nil {
		t.Fatalf("error creating new tab: %s\n", err)
	}

	if tabLen+1 != len(auto.tabs) {
		t.Fatalf("error created new tab but not reflected in our map")
	}
}

func TestCloseTab(t *testing.T) {
	var err error
	var newTab *Tab
	auto := testDefaultStartup(t)
	defer auto.Shutdown()

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
}

func testDefaultStartup(t *testing.T) *AutoGcd {
	s := NewSettings(testPath, testRandomDir(t))
	s.RemoveUserDir(true)
	s.AddStartupFlags(testStartupFlags)
	s.SetDebuggerPort(testRandomPort(t))
	auto := NewAutoGcd(s)
	if err := auto.Start(); err != nil {
		t.Fatalf("failed to start chrome: %s\n", err)
	}
	return auto
}

func testServer() {
	testListener, _ = net.Listen("tcp", ":0")
	_, testServerPort, _ := net.SplitHostPort(testListener.Addr().String())
	testServerAddr = fmt.Sprintf("http://localhost:%s/", testServerPort)
	go http.Serve(testListener, http.FileServer(http.Dir("testdata/")))
}

func testRandomPort(t *testing.T) string {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, randPort, _ := net.SplitHostPort(l.Addr().String())
	l.Close()
	return randPort
}

func testRandomDir(t *testing.T) string {
	dir, err := ioutil.TempDir(testDir, "autogcd")
	if err != nil {
		t.Fatalf("error getting temp dir: %s\n", err)
	}
	return dir
}
