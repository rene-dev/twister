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

// Usefil

package web

import (
	"io"
	"mime"
	"os"
	"path"
	"strconv"
	"strings"
)

// FileHandler returns a request handler that serves static files from root
// using using the relative request parameter "path". The "path" parameter is
// typically set using a Router pattern match:
//
//  r.Register("/static/<path:.*>", "GET", FileHandler(root))
//
// If the "v" paramter is supplied, then the cache control headers are set to expire
// the file in 10 years. 
func FileHandler(root string, extraHeader ...string) Handler {
	if !path.IsAbs(root) {
		wd, err := os.Getwd()
		if err != nil {
			panic("twister: FileHandler could not find cwd")
		}
		root = path.Join(wd, root)
	}
	root = path.Clean(root) + "/"
	return &fileHandler{root, extraHeader}
}

// fileHandler serves static files.
type fileHandler struct {
	root   string
	header []string
}

func (fh *fileHandler) ServeWeb(req *Request) {

	fname, found := req.Param.Get("path")
	if !found {
		panic("twister: FileHandler expects path param")
	}

	fname = path.Clean(fh.root + fname)
	if !strings.HasPrefix(fname, fh.root) {
		req.Error(StatusNotFound, os.NewError("twister: FileHandler access outside of root"))
		return
	}

	info, err := os.Stat(fname)
	if err != nil || !info.IsRegular() {
		req.Error(StatusNotFound, err)
		return
	}

	status := StatusOK
	etag := strconv.Itob64(info.Mtime_ns, 36)
	header := NewStringsMap(fh.header...)

	if i := strings.Index(req.Header.GetDef(HeaderETag, ""), etag); i >= 0 {
		status = StatusNotModified
	} else {
		header.Set(HeaderETag, etag)
		header.Set(HeaderContentLength, strconv.Itoa64(info.Size))
		ext := path.Ext(fname)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			header.Set(HeaderContentType, contentType)
		}
		_, found = req.Param.Get("v")
		if found {
			header.Set(HeaderExpires, FormatDeltaDays(3650))
			header.Set(HeaderCacheControl, "max-age=315360000")
		} else {
			header.Set(HeaderCacheControl, "public")
		}
	}

	f, err := os.Open(fname, os.O_RDONLY, 0)
	if err != nil {
		req.Error(StatusNotFound, err)
		return
	}
	defer f.Close()

	w := req.Responder.Respond(status, header)
	if req.Method != "HEAD" && status != StatusNotModified {
		io.Copy(w, f)
	}
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
