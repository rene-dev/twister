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

package web

import (
	"io"
	"mime"
	"os"
	"path"
	"strconv"
	"strings"
	"fmt"
	"expvar"
)

func serveFile(req *Request, fname string, extraHeader []string) {

	f, err := os.Open(fname, os.O_RDONLY, 0)
	if err != nil {
		req.Error(StatusNotFound, err)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || !info.IsRegular() {
		req.Error(StatusNotFound, err)
		return
	}

	status := StatusOK
	etag := strconv.Itob64(info.Mtime_ns, 36)
	header := NewStringsMap(extraHeader...)

	if i := strings.Index(req.Header.GetDef(HeaderIfNoneMatch, ""), etag); i >= 0 {
		status = StatusNotModified
	} else {
		header.Set(HeaderETag, etag)
		header.Set(HeaderContentLength, strconv.Itoa64(info.Size))
		ext := path.Ext(fname)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			header.Set(HeaderContentType, contentType)
		}
		if _, found := req.Param.Get("v"); found {
			header.Set(HeaderExpires, FormatDeltaDays(3650))
			header.Set(HeaderCacheControl, "max-age=315360000")
		} else {
			header.Set(HeaderCacheControl, "public")
		}
	}

	w := req.Responder.Respond(status, header)
	if req.Method != "HEAD" && status != StatusNotModified {
		io.Copy(w, f)
	}
}

// DirectoryHandler returns a request handler that serves static files from root
// using using the relative request parameter "path". The "path" parameter is
// typically set using a Router pattern match:
//
//  r.Register("/static/<path:.*>", "GET", DirectoryHandler(root))
//
// If the "v" request parameter is supplied, then the cache control headers are
// set to expire the file in 10 years. 
func DirectoryHandler(root string, extraHeader ...string) Handler {
	if !path.IsAbs(root) {
		wd, err := os.Getwd()
		if err != nil {
			panic("twister: DirectoryHandler could not find cwd")
		}
		root = path.Join(wd, root)
	}
	root = path.Clean(root) + "/"

	info, err := os.Stat(root)
	if err != nil || !info.IsDirectory() {
		panic("twister: root directory not found for DirectoryHandler.")
	}

	return &directoryHandler{root, extraHeader}
}

// directoryHandler serves static files from a directory.
type directoryHandler struct {
	root   string
	header []string
}

func (dh *directoryHandler) ServeWeb(req *Request) {

	fname, found := req.Param.Get("path")
	if !found {
		panic("twister: DirectoryHandler expects path param")
	}

	fname = path.Clean(dh.root + fname)
	if !strings.HasPrefix(fname, dh.root) {
		req.Error(StatusNotFound, os.NewError("twister: DirectoryHandler access outside of root"))
		return
	}

	serveFile(req, fname, dh.header)
}

// FileHandler returns a request handler that serves a static file specified by
// fname. 
//
// If the "v" request parameter is supplied, then the cache control headers are
// set to expire the file in 10 years. 
func FileHandler(fname string, extraHeader ...string) Handler {
	info, err := os.Stat(fname)
	if err != nil || !info.IsRegular() {
		panic("twister: file not found for FileHandler.")
	}
	return &fileHandler{fname, extraHeader}
}

// fileHandler servers static files.
type fileHandler struct {
	fname  string
	header []string
}

func (fh *fileHandler) ServeWeb(req *Request) {
	serveFile(req, fh.fname, fh.header)
}

type redirectHandler struct {
	url       string
	permanent bool
}

func (rh *redirectHandler) ServeWeb(req *Request) {
	req.Redirect(rh.url, rh.permanent)
}

// RedirectHandler returns a request handler that redirects to the given URL. 
func RedirectHandler(url string, permanent bool) Handler {
	return &redirectHandler{url, permanent}
}

var notFoundHandler = HandlerFunc(func(req *Request) { req.Error(StatusNotFound, nil) })

// NotFoundHandler returns a request handler that responds with 404 not found.
func NotFoundHandler() Handler {
	return notFoundHandler
}

// ExpvarHandler returns a handler that responds with the JSON encoding of
// variables exported with the expvar package.
func ExpvarHandler() Handler {
	return HandlerFunc(func(req *Request) {
		w := req.Respond(StatusOK, HeaderContentType, "application/json; charset=utf-8")
		fmt.Fprintf(w, "{\n")
		first := true
		for kv := range expvar.Iter() {
			if !first {
				fmt.Fprintf(w, ",\n")
			}
			first = false
			fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
		}
		fmt.Fprintf(w, "\n}\n")
	})
}
