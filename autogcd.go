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
to certain methods that require it. Internally a list of Documents is stored and will look up the element provided
a valid frameId is supplied.
*/
package autogcd

import (
	"github.com/wirepair/gcd"
	"sync"
)

type AutoGcd struct {
	debugger *gcd.Gcd
	settings *Settings
	tabLock  *sync.RWMutex
	tabs     map[string]*Tab
}

// Creates a new AutoGcd based off the provided settings.
func NewAutoGcd(settings *Settings) *AutoGcd {
	auto := &AutoGcd{settings: settings}
	auto.tabLock = &sync.RWMutex{}
	auto.tabs = make(map[string]*Tab)
	auto.debugger = gcd.NewChromeDebugger()

	if len(settings.extensions) > 0 {
		auto.debugger.AddFlags(settings.extensions)
	}

	if len(settings.flags) > 0 {
		auto.debugger.AddFlags(settings.flags)
	}

	if settings.timeout > 0 {
		auto.debugger.SetTimeout(settings.timeout)
	}

	return auto
}

// Starts Google Chrome with debugging enabled.
func (auto *AutoGcd) Start() error {
	var tabs []*gcd.ChromeTarget
	var err error
	auto.debugger.StartProcess(auto.settings.chromePath, auto.settings.userDir, auto.settings.chromePort)

	tabs, err = auto.debugger.GetTargets()
	if err != nil {
		return err
	}

	auto.tabLock.Lock()
	for _, tab := range tabs {
		auto.tabs[tab.Target.Id] = NewTab(tab)
	}
	auto.tabLock.Unlock()
	return nil
}

// Closes all tabs and shuts down the browser.
func (auto *AutoGcd) Shutdown() {
	auto.tabLock.Lock()
	for _, tab := range auto.tabs {
		auto.debugger.CloseTab(tab.ChromeTarget)
	}
	auto.tabLock.Unlock()
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

	tab := NewTab(target)
	auto.tabs[target.Target.Id] = tab
	return tab, nil
}

// Closes the provided tab.
func (auto *AutoGcd) CloseTab(tab *Tab) error {
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
