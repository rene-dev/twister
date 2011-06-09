// Copyright 2010 Gary Burd
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// The web package defines the application programming interface to a web
// server and implements functionality common to many web applications.
package web

import (
	"bufio"
	"http"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	ErrInvalidState          = os.NewError("object in invalid state")
	ErrBadFormat             = os.NewError("bad data format")
	ErrRequestEntityTooLarge = os.NewError("HTTP request entity too large")
)

// Responder represents the response.
type Responder interface {
	// Respond commits the status and headers to the network and returns
	// a writer for the response body.
	Respond(status int, header Header) (responseBody io.Writer)

	// Hijack lets the caller take over the connection from the HTTP server.
	// The caller is responsible for closing the connection. Returns connection
	// and bufio Reader with any data that might be buffered by the server.
	// Hijack is not supported by all servers.
	Hijack() (conn net.Conn, br *bufio.Reader, err os.Error)
}

// Request represents an HTTP request to the server.
type Request struct {
	// The response.
	Responder Responder

	// Uppercase request method. GET, POST, etc.
	Method string

	// The request URL with host and scheme set appropriately.
	URL *http.URL

	// Protocol version: major version * 1000 + minor version	
	ProtocolVersion int

	// The IP address of the client sending the request to the server.
	RemoteAddr string

	// Header maps canonical header names to slices of header values.
	Header Header

	// Request params from the query string, post body, routers and other.
	Param Values

	// Cookies.
	Cookie Values

	// Lowercase content type, not including params.
	ContentType string

	// Parameters from Content-Type header
	ContentParam map[string]string

	// ErrorHandler responds to the request with the given status code.
	// Applications can set the error handler using middleware. 
	ErrorHandler ErrorHandler

	// ContentLength is the length of the request body or -1 if the content
	// length is not known.
	ContentLength int

	// The request body.
	Body io.Reader

	// Attributes attached to the request by middleware. 
	Env map[string]interface{}
}

// ErrorHandler handles request errors.
type ErrorHandler func(req *Request, status int, reason os.Error, header Header)

// Handler is the interface for web handlers.
type Handler interface {
	ServeWeb(req *Request)
}

// HandlerFunc is a type adapter to allow the use of ordinary functions as web
// handlers. If the function returns an error, then the adapter responds to the
// request with an error response.
type HandlerFunc func(*Request)

// ServeWeb calls f(req).
func (f HandlerFunc) ServeWeb(req *Request) { f(req) }

// NewRequest allocates and initializes a request. This function is provided
// for the convenience of protocol adapters (fcgi, native http server, ...).
func NewRequest(remoteAddr string, method string, url *http.URL, protocolVersion int, header Header) (req *Request, err os.Error) {
	req = &Request{
		RemoteAddr:      remoteAddr,
		Method:          strings.ToUpper(method),
		URL:             url,
		ProtocolVersion: protocolVersion,
		ErrorHandler:    defaultErrorHandler,
		Param:           make(Values),
		Header:          header,
		Cookie:          make(Values),
		Env:             make(map[string]interface{}),
	}

	err = req.Param.ParseFormEncodedBytes([]byte(req.URL.RawQuery))
	if err != nil {
		return nil, err
	}

	err = parseCookieValues(header[HeaderCookie], req.Cookie)
	if err != nil {
		return nil, err
	}

	if s := req.Header.Get(HeaderContentLength); s != "" {
		var err os.Error
		req.ContentLength, err = strconv.Atoi(s)
		if err != nil {
			return nil, os.NewError("bad content length")
		}
	} else if method != "HEAD" && method != "GET" {
		req.ContentLength = -1
	}

	req.ContentType, req.ContentParam = req.Header.GetValueParam(HeaderContentType)
	return req, nil
}

// Respond is a convenience function that adds (key, value) pairs in
// headerKeysAndValues to a Header and calls through to the responder's
// Respond method.
func (req *Request) Respond(status int, headerKeysAndValues ...string) io.Writer {
	return req.Responder.Respond(status, NewHeader(headerKeysAndValues...))
}

func defaultErrorHandler(req *Request, status int, reason os.Error, header Header) {
	header.Set(HeaderContentType, "text/plain; charset=utf-8")
	w := req.Responder.Respond(status, header)
	io.WriteString(w, StatusText(status))
	if reason != nil || status >= 500 {
		log.Println("ERROR", req.URL, status, reason)
	}
}

// Error responds to the request with an error. 
func (req *Request) Error(status int, reason os.Error, headerKeysAndValues ...string) {
	req.ErrorHandler(req, status, reason, NewHeader(headerKeysAndValues...))
}

// Redirect responds to the request with a redirect to the specified URL.
func (req *Request) Redirect(url string, perm bool, headerKeysAndValues ...string) {
	status := StatusFound
	if perm {
		status = StatusMovedPermanently
	}

	// Make relative path absolute
	u, err := http.ParseURL(url)
	if err != nil && u.Scheme == "" && url[0] != '/' {
		d, _ := path.Split(req.URL.Path)
		url = d + url
	}

	header := NewHeader(headerKeysAndValues...)
	header.Set(HeaderLocation, url)
	req.Responder.Respond(status, header)
}

// BodyBytes returns the request body a slice of bytes. If maxLen is negative,
// then no limit is imposed on the length of the body. If the body is longer
// than maxLen, then ErrRequestEntityTooLarge is returned.
func (req *Request) BodyBytes(maxLen int) ([]byte, os.Error) {
	var p []byte

	if maxLen < 0 {
		maxLen = math.MaxInt32
	}

	if req.ContentLength == 0 {
		return nil, nil
	} else if req.ContentLength > maxLen {
		return nil, ErrRequestEntityTooLarge
	} else if req.ContentLength > 0 {
		p = make([]byte, req.ContentLength)
		if _, err := io.ReadFull(req.Body, p); err != nil {
			return nil, err
		}
	} else {
		var err os.Error
		lr := io.LimitedReader{R: req.Body, N: int64(maxLen) + 1}
		if p, err = ioutil.ReadAll(&lr); err != nil {
			return nil, err
		}
		if lr.N <= 0 {
			return nil, ErrRequestEntityTooLarge
		}
	}
	return p, nil
}

// ParseForm parses url-encoded form bodies. ParseForm is idempotent. Most
// applications should use the FormHandler middleware instead of calling this
// method directly.
func (req *Request) ParseForm(maxRequestBodyLen int) os.Error {
	if req.Env["twister.web.formparsed"] != nil ||
		req.ContentType != "application/x-www-form-urlencoded" ||
		req.ContentLength == 0 ||
		(req.Method != "POST" && req.Method != "PUT") {
		return nil
	}
	req.Env["twister.web.formparsed"] = true
	p, err := req.BodyBytes(maxRequestBodyLen)
	if err != nil {
		return err
	}
	if err := req.Param.ParseFormEncodedBytes(p); err != nil {
		return err
	}
	return nil
}

// Flusher is implemented by response bodies that allow the HTTP handler to
// flush buffered data to the network. Flush data to the network is useful for
// implementing long polling and other Comet mechanisms. 
type Flusher interface {
	Flush() os.Error
}
