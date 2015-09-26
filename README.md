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
The chrome debugger service uses internal nodeIds for identifying unique elements/nodes in the DOM. In most cases you will not need to use this identifier directly, however if you plan on calling gcdapi related features you will probably need to know the element's nodeId. The same is true for frameIds where they represent frames within a single page and can be nested. 


### Frames
If you need to search elements (by id or by a selector) of a frame's #document, you'll need to get an Element reference that is the iframe's #document. This can be done by doing a tab.GetElementsBySelector("iframe"), iterating over them and calling element.GetFrameDocumentNodeId(). This will return the internal document node id which you can then pass to tab.GetDocumentElementsBySelector(iframeDocNodeId, "#whatever").


### Stability & Waiting
There are a few ways you can test for stability or if an Element is ready. Element.WaitForReady() will not return until the debugger service has populated the element's information. If you are waiting for a page to stabilize, you can use the tab.WaitStable() method which won't return until it hasn't seen any DOM nodes being added/removed for a configurable (tab.SetStabilityTime(...)) amount of time. 

Finally, you can use the tab.WaitFor method, which takes a ConditionalFunc type and repeatedly calls it until it returns true, or times out.

For example/simple ConditionalFuncs see the [conditionals.go](https://github.com/wirepair/autogcd/tree/master/conditionals.go) source. Of course you can use whatever you want as long as it matches the ConditionalFunc prototype.