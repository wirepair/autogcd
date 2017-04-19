package autogcd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

var (
	testListener   net.Listener
	testPath       string
	testDir        string
	testPort       string
	testServerAddr string
)

var testStartupFlags = []string{"--test-type", "--ignore-certificate-errors", "--allow-running-insecure-content", "--disable-new-tab-first-run", "--no-first-run", "--disable-translate", "--safebrowsing-disable-auto-update", "--disable-component-update", "--safebrowsing-disable-download-protection"}

func init() {
	switch runtime.GOOS {
	case "windows":
		flag.StringVar(&testPath, "chrome", "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", "path to chrome")
		flag.StringVar(&testDir, "dir", "C:\\temp\\", "user directory")
	case "darwin":
		flag.StringVar(&testPath, "chrome", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "path to chrome")
		flag.StringVar(&testDir, "dir", "/tmp/", "user directory")
	case "linux":
		flag.StringVar(&testPath, "chrome", "/usr/bin/chromium-browser", "path to chrome")
		flag.StringVar(&testDir, "dir", "/tmp/", "user directory")
	}
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
	auto.SetTerminationHandler(nil)
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

func TestChromeTermination(t *testing.T) {
	auto := testDefaultStartup(t)
	doneCh := make(chan struct{})
	shutdown := time.NewTimer(time.Second * 4)
	timeout := time.NewTimer(time.Second * 10)
	terminatedHandler := func(reason string) {
		t.Logf("reason: %s\n", reason)
		doneCh <- struct{}{}
	}

	auto.SetTerminationHandler(terminatedHandler)
	for {
		select {
		case <-doneCh:
			goto DONE
		case <-shutdown.C:
			auto.Shutdown()
		case <-timeout.C:
			t.Fatalf("timed out waiting for termination")
		}
	}
DONE:
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
	auto.SetTerminationHandler(nil) // do not want our tests to panic
	return auto
}

func testServer() {
	testListener, _ = net.Listen("tcp", ":0")
	_, testServerPort, _ := net.SplitHostPort(testListener.Addr().String())
	testServerAddr = fmt.Sprintf("http://localhost:%s/", testServerPort)
	go http.Serve(testListener, http.FileServer(http.Dir("testdata")))
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

func testInstanceStartup(t *testing.T) (*AutoGcd, *exec.Cmd) {
	// ta := testDefaultStartup(t)
	port := testRandomPort(t)
	userDir := testRandomDir(t)
	flags := append(testStartupFlags, fmt.Sprintf("--remote-debugging-port=%s", port))
	flags = append(flags, fmt.Sprintf("--user-data-dir=%s", userDir))
	cmd := exec.Command(testPath, flags...)
	err := cmd.Start()
	if err != nil {
		log.Printf("start chrome ret err %+v", err)
		return nil, nil
	}
	s := NewSettings("", "")

	s.SetInstance("localhost", port)

	auto := NewAutoGcd(s)
	auto.Start()
	if err := auto.Start(); err != nil {
		t.Fatalf("failed to start chrome: %s\n", err)
	}
	auto.SetTerminationHandler(nil) // do not want our tests to panic
	return auto, cmd
}

func TestInstanceGetTab(t *testing.T) {
	var err error
	var tab *Tab
	auto, cmd := testInstanceStartup(t)
	defer func() { auto.Shutdown(); cmd.Process.Kill() }()

	tab, err = auto.GetTab()
	if err != nil {
		t.Fatalf("Error getting tab: %s\n", err)
	}

	if tab.Target.Type != "page" {
		t.Fatalf("Got tab but wasn't of type Page")
	}
}

func TestInstanceNewTab(t *testing.T) {
	//var newTab *Tab
	auto, cmd := testInstanceStartup(t)
	defer func() { auto.Shutdown(); cmd.Process.Kill() }()

	tabLen := len(auto.tabs)
	_, err := auto.NewTab()
	if err != nil {
		t.Fatalf("error creating new tab: %s\n", err)
	}

	if tabLen+1 != len(auto.tabs) {
		t.Fatalf("error created new tab but not reflected in our map")
	}
}

func TestInstanceCloseTab(t *testing.T) {
	var err error
	var newTab *Tab
	auto, cmd := testInstanceStartup(t)
	defer func() { auto.Shutdown(); cmd.Process.Kill() }()

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
