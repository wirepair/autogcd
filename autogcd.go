/*
The MIT License (MIT)

Copyright (c) 2016 isaac dawson

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

/*
Autogcd - An automation interface for https://github.com/wirepair/gcd. Contains most functionality
found in WebDriver and extends it to offer more low level features. This library was built due to
WebDriver/Chromedriver also using the debugger service. Since it is not possible to attach to a Page's
debugger twice, automating a custom extension with WebDriver turned out to not be possible.

The Chrome Debugger by nature is far more asynchronous than WebDriver. It is possible to work with
elements even though the debugger has not yet notified us of their existence. To deal with this, Elements
can be in multiple states; Ready, NotReady or Invalid. Only certain features are available when an Element
is in a Ready state. If an Element is Invalid, it should no longer be used and references to it should be
discarded.

Dealing with frames is also different than WebDriver. There is no SwitchToFrame, you simply pass in the frameId
to certain methods that require it. You can lookup the these frame documents by finding frame/iframe Elements and
requesting the document NodeId reference via the GetFrameDocumentNodeId method.

Lastly, dealing with windows... doesn't really work since they open a new tab. A possible solution would be to monitor
the list of tabs by calling AutoGcd.RefreshTabs() and doing a diff of known versus new. You could then do a Tab.Reload()
to refresh the page. It is recommended that you clear cache on the tab first so it is possible to trap the various
network events. There are other dirty hacks you could do as well, such as injecting script to override window.open,
or rewriting links etc.
*/
package autogcd

import (
	"errors"
	"fmt"
	"github.com/wirepair/gcd"
	"os"
	"sync"
)

type AutoGcd struct {
	debugger *gcd.Gcd
	settings *Settings
	tabLock  *sync.RWMutex
	tabs     map[string]*Tab
	shutdown bool
}

// Creates a new AutoGcd based off the provided settings.
func NewAutoGcd(settings *Settings) *AutoGcd {
	auto := &AutoGcd{settings: settings}
	auto.tabLock = &sync.RWMutex{}
	auto.tabs = make(map[string]*Tab)
	auto.debugger = gcd.NewChromeDebugger()
	auto.debugger.SetTerminationHandler(auto.defaultTerminationHandler)
	if len(settings.extensions) > 0 {
		auto.debugger.AddFlags(settings.extensions)
	}

	if len(settings.flags) > 0 {
		auto.debugger.AddFlags(settings.flags)
	}

	if settings.timeout > 0 {
		auto.debugger.SetTimeout(settings.timeout)
	}

	if len(settings.env) > 0 {
		auto.debugger.AddEnvironmentVars(settings.env)
	}
	return auto
}

// Default termination handling is to log, override with SetTerminationHandler
func (auto *AutoGcd) defaultTerminationHandler(reason string) {
	panic(fmt.Sprintf("chrome was terminated: %s\n", reason))
}

// Allow callers to handle chrome terminating.
func (auto *AutoGcd) SetTerminationHandler(handler gcd.TerminatedHandler) {
	auto.debugger.SetTerminationHandler(handler)
}

// Starts Google Chrome with debugging enabled.
func (auto *AutoGcd) Start() error {
	auto.debugger.StartProcess(auto.settings.chromePath, auto.settings.userDir, auto.settings.chromePort)
	tabs, err := auto.debugger.GetTargets()
	if err != nil {
		return err
	}
	auto.tabLock.Lock()
	for _, tab := range tabs {
		t, err := open(tab)
		if err != nil {
			return err
		}
		auto.tabs[tab.Target.Id] = t
	}
	auto.tabLock.Unlock()
	return nil
}

// Closes all tabs and shuts down the browser.
func (auto *AutoGcd) Shutdown() error {
	if auto.shutdown {
		return errors.New("AutoGcd already shut down.")
	}

	auto.tabLock.Lock()
	for _, tab := range auto.tabs {
		tab.close() // exit go routines
		auto.debugger.CloseTab(tab.ChromeTarget)

	}
	auto.tabLock.Unlock()
	err := auto.debugger.ExitProcess()
	if auto.settings.removeUserDir == true {
		return os.RemoveAll(auto.settings.userDir)
	}

	auto.shutdown = true
	return err
}

// Refreshs our internal list of tabs and return all tabs
func (auto *AutoGcd) RefreshTabList() (map[string]*Tab, error) {

	knownTabs := auto.GetAllTabs()
	knownIds := make(map[string]struct{}, len(knownTabs))
	for _, v := range knownTabs {
		knownIds[v.Target.Id] = struct{}{}
	}
	newTabs, err := auto.debugger.GetNewTargets(knownIds)
	if err != nil {
		return nil, err
	}

	auto.tabLock.Lock()
	for _, newTab := range newTabs {
		t, err := open(newTab)
		if err != nil {
			return nil, err
		}
		auto.tabs[newTab.Target.Id] = t
	}
	auto.tabLock.Unlock()
	return auto.GetAllTabs(), nil
}

// Returns the first "visual" tab.
func (auto *AutoGcd) GetTab() (*Tab, error) {
	auto.tabLock.RLock()
	defer auto.tabLock.RUnlock()
	for _, tab := range auto.tabs {
		if tab.Target.Type == "page" {
			return tab, nil
		}
	}
	return nil, &InvalidTabErr{Message: "no Page tab types found"}
}

// Returns a safe copy of tabs
func (auto *AutoGcd) GetAllTabs() map[string]*Tab {
	auto.tabLock.RLock()
	defer auto.tabLock.RUnlock()
	tabs := make(map[string]*Tab)
	for id, tab := range auto.tabs {
		tabs[id] = tab
	}
	return tabs
}

// Activate the tab in the chrome UI
func (auto *AutoGcd) ActivateTab(tab *Tab) error {
	return auto.debugger.ActivateTab(tab.ChromeTarget)
}

// Activate the tab in the chrome UI, by tab id
func (auto *AutoGcd) ActivateTabById(id string) error {
	tab, err := auto.tabById(id)
	if err != nil {
		return err
	}
	return auto.ActivateTab(tab)
}

// Creates a new tab
func (auto *AutoGcd) NewTab() (*Tab, error) {
	target, err := auto.debugger.NewTab()
	if err != nil {
		return nil, &InvalidTabErr{Message: "unable to create tab: " + err.Error()}
	}
	auto.tabLock.Lock()
	defer auto.tabLock.Unlock()

	tab, err := open(target)
	if err != nil {
		return nil, err
	}
	auto.tabs[target.Target.Id] = tab
	return tab, nil
}

// Closes the provided tab.
func (auto *AutoGcd) CloseTab(tab *Tab) error {
	tab.close() // kill listening go routines

	if err := auto.debugger.CloseTab(tab.ChromeTarget); err != nil {
		return err
	}

	auto.tabLock.Lock()
	defer auto.tabLock.Unlock()

	delete(auto.tabs, tab.Target.Id)
	return nil
}

// Closes a tab based off the tab id.
func (auto *AutoGcd) CloseTabById(id string) error {
	tab, err := auto.tabById(id)
	if err != nil {
		return err
	}
	auto.CloseTab(tab)
	return nil
}

// Finds the tab by its id.
func (auto *AutoGcd) tabById(id string) (*Tab, error) {
	auto.tabLock.RLock()
	tab := auto.tabs[id]
	auto.tabLock.RUnlock()
	if tab == nil {
		return nil, &InvalidTabErr{"unknown tab id " + id}
	}
	return tab, nil
}
