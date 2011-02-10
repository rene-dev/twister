## Overview

Twister is an HTTP server and framework for the [Go](http://golang.org/)
programming language.

## Documentation

Run [godoc](http://golang.org/cmd/godoc/) to read the documentation.

## Packages

* [web](twister/tree/master/web) - Defines the application interface to a server and includes functionality used by most web applications.
  * Routing to handlers using regular expression match on host and path.
  * Protection against cross site request forgery.
  * Extensible model for middleware.
  * Cookie parsing.
  * WebSockets.
  * Static file handling.
  * Signed values for cookies and form parameters.
  * Multipart forms.
* [server](twister/tree/master/server) - An HTTP server impelemented in Go.
* [oauth](twister/tree/master/oauth) - OAuth client.
* [expvar](twister/tree/master/expvar) - Exports variables as JSON over HTTP for monitoring.
* [websocket](twister/tree/master/websocket) - WebSocket server implementation.

## Examples

* [examples/wiki](twister/tree/master/examples/wiki) - The [Go web application example](http://golang.org/doc/codelab/wiki/) converted to use Twister instead of the Go http package.
* [examples/demo](twister/tree/master/examples/demo) - Illustrates basic features of Twister.
* [examples/twitter](twister/tree/master/examples/twitter) - Login to Twitter with OAuth and display home timeline.
* [examples/facebook](twister/tree/master/examples/facebook) - Login to Facebook with OAuth2 and display news feed.

## Installation

1. [Install Go](http://golang.org/doc/install.html).
3. `goinstall github.com/garyburd/twister/server`

The Go distribution is Twister's only dependency.

## Feedback 

Feedback, comments and quesions are welcome. Send the Gary Burd a message
through [Github](https://github.com/inbox/new/garyburd) or
[Google](http://www.google.com/profiles/100190655365702878730/contactme).

## About

Twister was written by [Gary Burd](http://gary.beagledreams.com/). The name
"Twister" was inspired by [Tornado](http://tornadoweb.org/").

