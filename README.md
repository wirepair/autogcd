# Automating Google Chrome Debugger (autogcd)
Autogcd is a wrapper around the [gcd](https://github.com/wirepair/gcd/) library to enable automation of Google Chrome. It some what mimics the functionality offered by WebDriver but allows more low level access via the debugger service. 

## Changelog (2018)
- Feburary 22nd: Updated to latest gcd / protocol.json file for 66.0.3346.8. Added dep init/dep ensure to repository for package management.
- January 17th: Updated to latest gcd / protocol.json file for 65.0.3322.3.

## Changelog (2017)
- November 20th: Updated to latest gcd / protocol.json file for 64.0.3269.3. Navigate now exposes friendly error text.
- October 30th: Updated to latest gcd / protocol.json file for 64.0.3251.0.
- September 9th: Updated to latest gcd / protocol.json file for 62.0.3202.9. We are now bound to the browser version (latest dev channel). See [gcd](https://github.com/wirepair/gcd/) updates for more information. Note if you change the gcd channel, you'll probably end up breaking a lot of methods in autogcd, so be warned.
- August 15th: Updated to the latest gcd / protocol.json file.
- May: Updated to latest gcd / protocol.json file.
- April: Updated to latest gcd / protocol.json file. Fixed unit tests to wait for stability a bit longer as node changes seem to take longer than before.

## Dependencies
autogcd requires [gcd](https://github.com/wirepair/gcd/), [gcdapi](https://github.com/wirepair/gcd/tree/master/gcdapi) and [gcdmessage](https://github.com/wirepair/gcd/tree/master/gcdmessage) packages. 

## The API
Autogcd is comprised of four components:
* [autogcd.go](https://github.com/wirepair/autogcd/tree/master/autogcd.go) - The wrapper around gcd.Gcd. 
* [settings.go](https://github.com/wirepair/autogcd/tree/master/settings.go) - For managing startup of autogcd.
* [tab.go](https://github.com/wirepair/autogcd/tree/master/tab.go) - Individual chrome tabs
* [element.go](https://github.com/wirepair/autogcd/tree/master/element.go) - Elements that make up the page (includes iframes, #documents as well)

## API Documentation
[Documentation](https://godoc.org/github.com/wirepair/autogcd/)

## Usage
See the [examples](https://github.com/wirepair/autogcd/tree/master/examples) or the various testcases.

## Notes
The chrome debugger service uses internal nodeIds for identifying unique elements/nodes in the DOM. In most cases you will not need to use this identifier directly, however if you plan on calling gcdapi related features you will probably need it. The most common example of when you'll need them is for getting access to a nested #document element inside of an iframe. To run query selectors on nested documents, the nodeId of the iframe #document must be known.

### Elements
The Chrome Debugger by nature is far more asynchronous than WebDriver. It is possible to work with elements even though the debugger has not yet notified us of their existence. To deal with this, Elements can be in multiple states; Ready, NotReady or Invalid. Only certain features are available when an Element is in a Ready state. If an Element is Invalid, it should no longer be used and references to it should be discarded.

### Frames
If you need to search elements (by id or by a selector) of a frame's #document, you'll need to get an Element reference that is the iframe's #document. This can be done by doing a tab.GetElementsBySelector("iframe"), iterating over the results and calling element.GetFrameDocumentNodeId(). This will return the internal document node id which you can then pass to tab.GetDocumentElementsBySelector(iframeDocNodeId, "#whatever").

### Windows
The major limitation of using the Google Chrome Remote Debugger is when working with windows. Since each tab must have the debugger enabled, calls to window.open will open a new window prior to us being able to attach a debugger. To get around this, you'll need to get a list of tabs AutoGcd.GetAllTabs(), then call AutoGcd.RefreshTabList() which will connect each tab to an autogcd.Tab. You'd then need to reload the tab get begin working with it. 

### Stability & Waiting
There are a few ways you can test for stability or if an Element is ready. Element.WaitForReady() will not return until the debugger service has populated the element's information. If you are waiting for a page to stabilize, you can use the tab.WaitStable() method which won't return until it hasn't seen any DOM nodes being added/removed for a configurable (tab.SetStabilityTime(...)) amount of time. 

Finally, you can use the tab.WaitFor method, which takes a ConditionalFunc type and repeatedly calls it until it returns true, or times out.

For example/simple ConditionalFuncs see the [conditionals.go](https://github.com/wirepair/autogcd/tree/master/conditionals.go) source. Of course you can use whatever you want as long as it matches the ConditionalFunc signature.

### Navigation Errors
Unlike WebDriver, we can determine if navigation fails, *at least in chromium. After tab.Navigate(url), calling tab.DidNavigationFail() will return a true/false return value along with a string of the failure type if one did occur. It is strongly recommended you pass the following flags: --test-type, --ignore-certificate-errors on start up of autogcd if you wish to ignore certificate errors.

\* This does not appear to work in chrome in windows or osx.

### Input
Only a limited set of input functions have been implemented. Clicking and sending keys. You can use Element.SendKeys() or send the keys to whatever is focused by using Tab.SendKeys(). Only Enter ("\n"), Tab ("\t") and Backspace ("\b") were implemented, to use them, simply add them to your SendKeys argument Element.SendKeys("enter text hit enter\n") where \n will cause the enter key to be pressed. 

### Listeners
Four listener functions have been implemented, GetConsoleMessages, GetNetworkTraffic, GetStorageEvents, GetDOMChanges. 

#### GetConsoleMessages 
Pass in a ConsoleMessageFunc handler to begin receiving console messages from the tab. Use StopConsoleMessages to stop receiving them.

#### GetNetworkTraffic
Pass in either a NetworkRequestHandlerFunc, NetworkResponseHandlerFunc or NetworkFinishedHandlerFunc handler (or all three) to receive network traffic events. NetworkFinishedHandler should be used to signal your application that it's safe to get the response body of the request. While calling GetResponseBody *may* work from NetworkResponseHandlerFunc, it will in many cases fail as the debugger service isn't ready to return the data yet. Use StopNetworkTraffic to stop receiving them.

#### GetStorageEvents
Pass in a StorageFunc handler to recieve cleared, removed, added and updated storage events. Use StopStorageEvents to stop receiving them.

#### GetDOMChanges
Pass in a DomChangeHandlerFunc to receive various dom change events. Call it with a nil handler to stop receiving them.

## Calling gcd directly
AutoGcd has not implemented all of the Google Chrome Debugger protocol methods and features because I don't see any point in wrapping a lot of them. However, you are not out of luck, all gcd components are bound to each Tab object. I'd suggest reviewing the gcdapi package if there is a particular component you wish to use. All components are bound to the Tab so it should be as simple as calling Tab.{component}.{method}.

### Overriding gcd
Take a look at [api_overrides.go](https://github.com/wirepair/autogcd/tree/master/api_overrides.go) for an example of overriding gcd methods. In
some cases the protocol.json specification is incorrect, in which case you may need to override specific methods. Since I designed the packages
to use an intermediary gcdmessage package for requests and responses you're completely free to override anything necessary. 

## Internals
I'll admit, I do not fully like the design of the Elements. I have to track state updates very carefully and I chose to use sync.RWMutex locks. I couldn't see an obvious method of using channels to synchronize access to the DOMNodes. I'm very open to new architectures/designs if someone has a better method of keeping Element objects up to date as Chrome notifies autogcd of new values. 

As mentioned in the Elements section, Chrome Debugger Protocol is fully asynchronous. The debugger is only notified of elements when the page first loads (and even then only a few of the top level elements). It also occurs when an element has been modified, or when you request them with DOM.requestChildNodes. Autogcd tries to manage all of this for you, but there may be a case where you search for elements that chrome has not notified the debugger client yet. In this case the Element will be, in autogcd terminology, NotReady. This means you can sort of work with it because we know its nodeId but we may not know much else (even what type of node it is). Internally almost all chrome debugger methods take nodeIds. 

This package has been *heavily* tested in the real world. It was used to scan the top 1 million websites from Alexa. I found numerous goroutine leaks that have been subsequently fixed. After running my scan I no longer see any leaks. It should also be completely safe to kill the browser at any point and not have any runaway go routines since I have channels waiting for close messages at any point a channel is sending or receiving. 

## Reporting Bugs & Requesting Features
Found a bug? Great! Tell me what version of chrome/chromium you are using and how to reproduce and I'll get to it when I can. Keep in mind this is a side project for me. Same goes for new features. Patches obviously welcome. 


## License
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