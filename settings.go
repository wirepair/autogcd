/*
The MIT License (MIT)

Copyright (c) 2015 isaac dawson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

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

// Can really only be localhost, but this may change in the future so support it anyways.
func (s *Settings) SetChromeHost(host string) {
	s.chromeHost = host
}

// Sets the chrome debugger port.
func (s *Settings) SetDebuggerPort(chromePort string) {
	s.chromePort = chromePort
}

// How long to wait for chrome to startup and allow us to connect.
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
// the chrome.debugger API since you can not attach debuggers to a Tab twice.
func (s *Settings) AddExtension(paths []string) {
	for _, ext := range paths {
		s.extensions = append(s.extensions, fmt.Sprintf("--load-extension=%s", ext))
	}
}
