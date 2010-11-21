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
	"bytes"
	"http"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"math"
)

var (
	// Object not in valid state for call.
	ErrInvalidState          = os.NewError("invalid state")
	ErrBadFormat             = os.NewError("bad format")
	ErrRequestEntityTooLarge = os.NewError("request entity too large")
)

// StringsMap maps strings to slices of strings.
type StringsMap map[string][]string

// NewStringsMap returns a map initialized with the given key-value pairs.
func NewStringsMap(kvs ...string) StringsMap {
	if len(kvs)%2 == 1 {
		panic("twister: even number args required for NewStringsMap")
	}
	m := make(StringsMap)
	for i := 0; i < len(kvs); i += 2 {
		m.Append(kvs[i], kvs[i+1])
	}
	return m
}

// Get returns the first value for given key or "" if the key is not found.
func (m StringsMap) Get(key string) (value string, found bool) {
	values, found := m[key]
	if !found || len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetDef returns first value for given key, or def if the key is not found.
func (m StringsMap) GetDef(key string, def string) string {
	values, found := m[key]
	if !found || len(values) == 0 {
		return def
	}
	return values[0]
}

// Append value to slice for given key.
func (m StringsMap) Append(key string, value string) {
	m[key] = append(m[key], value)
}

// Set value for given key, discarding previous values if any.
func (m StringsMap) Set(key string, value string) {
	m[key] = []string{value}
}

// StringMap returns a string to string map by discarding all but the first
// value for a key.
func (m StringsMap) StringMap() map[string]string {
	result := make(map[string]string)
	for key, values := range m {
		result[key] = values[0]
	}
	return result
}

// FormEncode returns a buffer containing the URL form encoding of the map.
func (m StringsMap) FormEncode() []byte {
	var b bytes.Buffer
	sep := false
	for key, values := range m {
		escapedKey := http.URLEscape(key)
		for _, value := range values {
			if sep {
				b.WriteByte('&')
			} else {
				sep = true
			}
			b.WriteString(escapedKey)
			b.WriteByte('=')
			b.WriteString(http.URLEscape(value))
		}
	}
	return b.Bytes()
}

// RequestBody represents the request body.
type RequestBody interface {
	io.Reader
}

// ResponseBody represents the response body.
type ResponseBody interface {
	io.Writer
	// Flush writes any buffered data to the network.
	Flush() os.Error
}

// Responder represents the the response.
type Responder interface {
	// Respond commits the status and headers to the network and returns
	// a writer for the response body.
	Respond(status int, header StringsMap) ResponseBody

	// Hijack lets the caller take over the connection from the HTTP server.
	// The caller is responsible for closing the connection. Returns connection
	// and bytes buffered by the server.
	Hijack() (conn net.Conn, buf []byte, err os.Error)
}

// Request represents an HTTP request.
type Request struct {
	Responder Responder // The response.

	// Uppercase request method. GET, POST, etc.
	Method string

	// The request URL with host and scheme set appropriately.
	URL *http.URL

	// Protocol version: major version * 1000 + minor version	
	ProtocolVersion int

	// The IP address of the client sending the request to the server.
	RemoteAddr string

	// Header maps canonical header names to slices of header values.
	Header StringsMap

	// Request params from the query string, post body, routers and other.
	Param StringsMap

	// Cookies.
	Cookie StringsMap

	// Lowercase content type, not including params.
	ContentType string

	// Parameters from Content-Type header
	ContentParam map[string]string

	// ErrorHandler responds to the request with the given status code.
	// Applications set their error handler in middleware. 
	ErrorHandler func(req *Request, status int, reason os.Error)

	// ContentLength is the length of the request body or -1 if the content
	// length is not known.
	ContentLength int

	// The request body.
	Body RequestBody

	formParsed bool
}

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

// NewRequest allocates and initializes a request. 
func NewRequest(remoteAddr string, method string, url *http.URL, protocolVersion int, header StringsMap) (req *Request, err os.Error) {
	req = &Request{
		RemoteAddr:      remoteAddr,
		Method:          strings.ToUpper(method),
		URL:             url,
		ProtocolVersion: protocolVersion,
		ErrorHandler:    defaultErrorHandler,
		Param:           make(StringsMap),
		Header:          header,
		Cookie:          make(StringsMap),
	}

	err = ParseUrlEncodedFormBytes([]byte(req.URL.RawQuery), req.Param)
	if err != nil {
		return nil, err
	}

	err = parseCookieValues(header[HeaderCookie], req.Cookie)
	if err != nil {
		return nil, err
	}

	if s, found := req.Header.Get(HeaderContentLength); found {
		var err os.Error
		req.ContentLength, err = strconv.Atoi(s)
		if err != nil {
			return nil, os.NewError("bad content length")
		}
	} else if method != "HEAD" && method != "GET" {
		req.ContentLength = -1
	}

	if s, found := req.Header.Get(HeaderContentType); found {
		req.ContentType, req.ContentParam = mime.ParseMediaType(s)
	}

	return req, nil
}

// Respond is a convenience function that adds (key, value) pairs in kvs to a
// StringsMap and calls through to the connection's Respond method.
func (req *Request) Respond(status int, kvs ...string) ResponseBody {
	return req.Responder.Respond(status, NewStringsMap(kvs...))
}

func defaultErrorHandler(req *Request, status int, reason os.Error) {
	w := req.Respond(status, HeaderContentType, "text/plain; charset=utf-8")
	io.WriteString(w, StatusText(status))
	log.Println("ERROR", req.URL, status, reason)
}

// Error responds to the request with an error. 
func (req *Request) Error(status int, reason os.Error) {
	req.ErrorHandler(req, status, reason)
}

// Redirect responds to the request with a redirect the specified URL.
func (req *Request) Redirect(url string, perm bool, kvs ...string) {
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

	header := NewStringsMap(kvs...)
	header.Set(HeaderLocation, url)
	req.Responder.Respond(status, header)
}

// BodyBytes returns the request body a slice of bytees. If maxLen is negative,
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
		if p, err = ioutil.ReadAll(io.LimitReader(req.Body, int64(maxLen))); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// ParseForm parses url-encoded form bodies. ParseForm is idempotent.
func (req *Request) ParseForm(maxRequestBodyLen int) os.Error {
	if req.formParsed ||
		req.ContentType != "application/x-www-form-urlencoded" ||
		req.ContentLength == 0 ||
		(req.Method != "POST" && req.Method != "PUT") {
		return nil
	}
	req.formParsed = true
	p, err := req.BodyBytes(maxRequestBodyLen)
	if err != nil {
		return err
	}
	if err := ParseUrlEncodedFormBytes(p, req.Param); err != nil {
		return err
	}
	return nil
}
