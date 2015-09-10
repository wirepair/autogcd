package autogcd

import (
	"fmt"
	"time"
)

type Settings struct {
	timeout    time.Duration
	chromePath string
	chromeHost string
	chromePort string
	userDir    string
	extensions []string
	flags      []string
}

func NewSettings(chromePath, userDir string) *Settings {
	s := &Settings{}
	s.chromePath = chromePath
	s.chromePort = "9222"
	s.userDir = userDir
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

func (s *Settings) AddStartupFlags(flags []string) {
	s.flags = append(s.flags, flags...)
}

func (s *Settings) AddExtension(path string) {
	s.extensions = append(s.extensions, fmt.Sprintf("--load-extension=%s", path))
}
