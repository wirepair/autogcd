package autogcd

import (
	"fmt"
	"time"
)

type Settings struct {
	timeout       time.Duration
	chromePath    string   // path to chrome
	chromeHost    string   // can really only be localhost
	chromePort    string   // port to chrome debugger
	userDir       string   // the user directory to use
	removeUserDir bool     // should we delete the user directory on shutdown?
	extensions    []string // custom extensions to load
	flags         []string // custom os.Environ flags to use to start the chrome process
}

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

func (s *Settings) AddStartupFlags(flags []string) {
	s.flags = append(s.flags, flags...)
}

func (s *Settings) AddExtension(path string) {
	s.extensions = append(s.extensions, fmt.Sprintf("--load-extension=%s", path))
}
