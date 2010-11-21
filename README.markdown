## Overview

Twister is an HTTP server for the [Go](http://golang.org/) programming language.

Twister is a work in progress. 

## Packages

* twister/web - Defines the application interface to a server and includes functionality used by most web applications.
  * Routing to handlers using regular expression match on path
  * Routing to handlers using host
  * Protection against cross site request forgery
  * Extensible model for middleware
  * Cookie parsing
  * WebSockets
  * Static file handling
  * Signed values for cookies and form parameters
  * Multipart forms
* twister/server - An HTTP server impelemented in Go.
* twister/oauth - OAuth client

## Examples

* twister/examples/demo - Illustrates basic features of Twister.
* twister/examples/twitter - Login to Twitter with OAuth and display home timeline.
* twister/examples/facebook - Login to Facebook with OAuth2 and display news feed.

## Installation

1. [Install Go](http://golang.org/doc/install.html).
2. `goinstall github.com/garyburd/twister/web`
2. `goinstall github.com/garyburd/twister/server`

## About

Twister was written by [Gary Burd](http://gary.beagledreams.com/). The name
"Twister" was inspired by [Tornado](http://tornadoweb.org/").

