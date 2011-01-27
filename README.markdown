## Overview

Twister is an HTTP server and framework for the [Go](http://golang.org/) programming language.

## Packages

* twister/web - Defines the application interface to a server and includes functionality used by most web applications.
  * Routing to handlers using regular expression match on host and path.
  * Protection against cross site request forgery.
  * Extensible model for middleware.
  * Cookie parsing.
  * WebSockets.
  * Static file handling.
  * Signed values for cookies and form parameters.
  * Multipart forms.
* twister/server - An HTTP server impelemented in Go.
* twister/oauth - OAuth client.
* twister/expvar - Exports variables as JSON over HTTP for monitoring.
* twister/websocket - WebSocket server implementation.

## Examples

* twister/examples/wiki - The [Go web application example](http://golang.org/doc/codelab/wiki/) converted to use Twister instead of the Go http package.
* twister/examples/demo - Illustrates basic features of Twister.
* twister/examples/twitter - Login to Twitter with OAuth and display home timeline.
* twister/examples/facebook - Login to Facebook with OAuth2 and display news feed.

## Installation

1. [Install Go](http://golang.org/doc/install.html).
3. `goinstall github.com/garyburd/twister/server`

The Go distribution is Twister's only dependency.

## About

Twister was written by [Gary Burd](http://gary.beagledreams.com/). The name
"Twister" was inspired by [Tornado](http://tornadoweb.org/").

