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