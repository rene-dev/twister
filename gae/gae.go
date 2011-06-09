// Copyright 2011 Gary Burd
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

// The gae package provides support for running Twister applications on
// Google AppEngine.
package gae

import (
	"appengine"
	"bufio"
	"github.com/garyburd/twister/web"
	"http"
	"net"
	"os"
	"io"
)

type responder struct{ w http.ResponseWriter }

func (r responder) Respond(status int, header web.Header) io.Writer {
	for k, v := range header {
		r.w.Header()[k] = v
	}
	r.w.WriteHeader(status)
	return r.w
}

func (r responder) Hijack() (conn net.Conn, br *bufio.Reader, err os.Error) {
	return nil, nil, os.NewError("not implemented")
}

func webRequestFromHTTPRequest(w http.ResponseWriter, r *http.Request) *web.Request {
	header := web.Header(map[string][]string(r.Header))
	foo := header.Get("Cookie")

	if r.Referer != "" {
		header.Set(web.HeaderReferer, r.Referer)
	}
	if r.UserAgent != "" {
		header.Set(web.HeaderUserAgent, r.UserAgent)
	}

	req, _ := web.NewRequest(
		r.RemoteAddr,
		r.Method,
		r.URL,
		web.ProtocolVersion(r.ProtoMajor, r.ProtoMinor),
		header)

	req.Body = r.Body
	req.Responder = responder{w}
	req.ContentLength = int(r.ContentLength)
	if r.Form != nil {
		req.Param = web.Values(map[string][]string(r.Form))
	}

	for _, c := range r.Cookie {
		req.Cookie.Add(c.Name, c.Value)
	}

	req.Env["foo"] = foo

	return req
}

type wrappedHandler struct{ h web.Handler }

func (h wrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := webRequestFromHTTPRequest(w, r)
	req.Env["twister.gae.context"] = appengine.NewContext(r)
	req.Env["twister.gae.request"] = r
	h.h.ServeWeb(req)
}

// Handle registers handler for the given URL pattern. The documentation for
// http.ServeMux explains how patterns are matched. Twister applicatons will
// typically register a handler with the pattern "/".
func Handle(pattern string, handler web.Handler) {
	http.Handle(pattern, wrappedHandler{handler})
}

// Context returns the App Engine context for the given request.
func Context(req *web.Request) appengine.Context {
	return req.Env["twister.gae.context"].(appengine.Context)
}
