# Automating Google Chrome Debugger (autogcd)
Autogcd is a wrapper around the [gcd](https://github.com/wirepair/gcd/) library to enable automation of Google Chrome. It some what mimics the functionality offered by WebDriver but allows more low level access via the debugger service. 

## Dependencies
autogcd requires [gcd](https://github.com/wirepair/gcd/), [gcdapi](https://github.com/wirepair/gcd/tree/master/gcdapi) and [gcdmessage](https://github.com/wirepair/gcd/tree/master/gcdmessage) packages. 

## The API
Autogcd is comprised of four components:
* autogcd - The wrapper around gcd.Gcd. 
* settings - For managing startup of autogcd.
* Tab - Individual chrome tabs
* Element - Elements that make up the Page

## Documentation
[Documentation](https://godoc.org/github.com/wirepair/autogcd/)

## Usage
See the [examples](https://github.com/wirepair/autogcd/tree/master/examples) or the various testcases.


## Notes
The chrome debugger service uses internal nodeIds for identifying unique elements/nodes in the DOM. In most cases you will not need to use this identifier directly, however if you plan on calling gcdapi related features you will probably need it. The most common example of when you'll need them is for getting access to a nested #document element inside of an iframe. To run query selectors on nested documents, the nodeId of the iframe #document must be known.


### Frames
If you need to search elements (by id or by a selector) of a frame's #document, you'll need to get an Element reference that is the iframe's #document. This can be done by doing a tab.GetElementsBySelector("iframe"), iterating over them and calling element.GetFrameDocumentNodeId(). This will return the internal document node id which you can then pass to tab.GetDocumentElementsBySelector(iframeDocNodeId, "#whatever").

### Windows
The major limitation of using the Google Chrome Remote Debugger is when working with windows. Since each tab must have the debugger enabled, calls to window.open will open a new window prior to us being able to attach a debugger. To get around this, you'll need to get a list of tabs AutoGcd.GetAllTabs(), then call AutoGcd.RefreshTabList() which will connect each tab to an autogcd.Tab. You'd then need to reload the tab get begin working with it. 

### Stability & Waiting
There are a few ways you can test for stability or if an Element is ready. Element.WaitForReady() will not return until the debugger service has populated the element's information. If you are waiting for a page to stabilize, you can use the tab.WaitStable() method which won't return until it hasn't seen any DOM nodes being added/removed for a configurable (tab.SetStabilityTime(...)) amount of time. 

Finally, you can use the tab.WaitFor method, which takes a ConditionalFunc type and repeatedly calls it until it returns true, or times out.

For example/simple ConditionalFuncs see the [conditionals.go](https://github.com/wirepair/autogcd/tree/master/conditionals.go) source. Of course you can use whatever you want as long as it matches the ConditionalFunc prototype.


### Input
Only a limited set of input functions have been implemented. Clicking and sending keys. You can use Element.SendKeys() or send the keys to whatever is focused by using Tab.SendKeys(). Only Enter ("\n"), Tab ("\t") and Backspace ("\b") were implemented, to use them, simply add them to your SendKeys argument Element.SendKeys("enter text hit enter\n") where \n will cause the enter key to be pressed. 

### Listeners
Four listener functions have been implemented, GetConsoleMessages, GetNetworkTraffic, GetStorageEvents, GetDOMChanges. 

#### GetConsoleMessages 
Pass in a ConsoleMessageFunc handler to begin receiving console messages from the tab. Use StopConsoleMessages to stop receiving them.

#### GetNetworkTraffic
Pass in either a NetworkRequestHandlerFunc, NetworkResponseHandlerFunc or NetworkFinishedHandlerFunc handler (or all three) to receive network traffic events. NetworkFinishedHandler's should be used to signal your application that it's safe to get the response body of the request. While calling GetResponseBody *may* work from NetworkResponseHandlerFunc, it will in many cases fail as the debugger service isn't ready to return the data yet. Use StopNetworkTraffic to stop receiving them.

#### GetStorageEvents
Pass in a StorageFunc handler to recieve cleared, removed, added and updated storage events. Use StopStorageEvents to stop receiving them.

#### GetDOMChanges
Pass in a DomChangeHandlerFunc to receive various dom change events. Call it with a nil handler to stop receiving them.