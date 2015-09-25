package autogcd

import (
	"fmt"
	"time"
)

type Settings struct {
	timeout       time.Duration // timeout for giving up on chrome starting and connecting to the debugger service
	chromePath    string        // path to chrome
	chromeHost    string        // can really only be localhost
	chromePort    string        // port to chrome debugger
	userDir       string        // the user directory to use
	removeUserDir bool          // should we delete the user directory on shutdown?
	extensions    []string      // custom extensions to load
	flags         []string      // custom os.Environ flags to use to start the chrome process
}

// Creates a new settings object to start Chrome and enable remote debugging
func NewSettings(chromePath, userDir string) *Settings {
	s := &Settings{}
	s.chromePath = chromePath
	s.chromePort = "9222"
	s.userDir = userDir
	s.removeUserDir = false
	s.extensions = make([]string, 0)
	s.flags = make([]string, 0)
	return s
}

func (s *Settings) SetChromeHost(host string) {
	s.chromeHost = host
}

func (s *Settings) SetDebuggerPort(chromePort string) {
	s.chromePort = chromePort
}

func (s *Settings) SetStartTimeout(timeout time.Duration) {
	s.timeout = timeout
}

// On Shutdown, deletes the userDir and files if true.
func (s *Settings) RemoveUserDir(shouldRemove bool) {
	s.removeUserDir = true
}

// Adds custom flags when starting the chrome process
func (s *Settings) AddStartupFlags(flags []string) {
	s.flags = append(s.flags, flags...)
}

// Adds a custom extension to launch with chrome. Note this extension MAY NOT USE
// the chrome.debugger API since you can not attach to a Tab twice with debuggers.
func (s *Settings) AddExtension(path string) {
	s.extensions = append(s.extensions, fmt.Sprintf("--load-extension=%s", path))
}
