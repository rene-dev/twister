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
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"log"
	"time"
)

type filterResponder struct {
	Responder
	filter func(status int, header HeaderMap) (int, HeaderMap)
}

func (rf *filterResponder) Respond(status int, header HeaderMap) ResponseBody {
	return rf.Responder.Respond(rf.filter(status, header))
}

// FilterRespond replaces the request's responder with one that filters the
// arguments to Respond through the supplied filter. This function is intended
// to be used by middleware.
func FilterRespond(req *Request, filter func(status int, header HeaderMap) (int, HeaderMap)) {
	req.Responder = &filterResponder{req.Responder, filter}
}

// SetErrorHandler returns a handler that sets the request's error handler to the supplied handler.
func SetErrorHandler(errorHandler ErrorHandler, handler Handler) Handler {
	return HandlerFunc(func(req *Request) {
		req.ErrorHandler = errorHandler
		handler.ServeWeb(req)
	})
}

// Name of XSRF cookie and request parameter.
const (
	XSRFCookieName = "xsrf"
	XSRFParamName  = "xsrf"
)

// ProcessForm returns a handler that parses URL encoded forms if smaller than the 
// specified size and optionally checks for XSRF.
func ProcessForm(maxRequestBodyLen int, checkXSRF bool, handler Handler) Handler {
	return HandlerFunc(func(req *Request) {

		if err := req.ParseForm(maxRequestBodyLen); err != nil {
			status := StatusBadRequest
			if err == ErrRequestEntityTooLarge {
				status = StatusRequestEntityTooLarge
				if _, found := req.Header.Get(HeaderExpect); found {
					status = StatusExpectationFailed
				}
			}
			req.Error(status, os.NewError("twister: Error reading or parsing form."))
			return
		}

		if checkXSRF {
			const tokenLen = 8
			expectedToken, found := req.Cookie.Get(XSRFCookieName)

			// Create new XSRF token?
			if !found || len(expectedToken) != tokenLen {
				p := make([]byte, tokenLen/2)
				_, err := rand.Reader.Read(p)
				if err != nil {
					panic("twister: rand read failed")
				}
				expectedToken = hex.EncodeToString(p)
				c := fmt.Sprintf("%s=%s; Path=/; HttpOnly", XSRFCookieName, expectedToken)
				FilterRespond(req, func(status int, header HeaderMap) (int, HeaderMap) {
					header.Append(HeaderSetCookie, c)
					return status, header
				})
			}

			actualToken := req.Param.GetDef(XSRFParamName, "")
			if expectedToken != actualToken {
				req.Param.Set(XSRFParamName, expectedToken)
				if req.Method == "POST" || req.Method == "PUT" {
					err := os.NewError("twister: bad xsrf token")
					if actualToken == "" {
						err = os.NewError("twister: missing xsrf token")
					}
					req.Error(StatusNotFound, err)
					return
				}
			}
		}

		handler.ServeWeb(req)
	})
}

func writeStringMap(w io.Writer, title string, m map[string][]string) {
	first := true
	for key, values := range m {
		if first {
			fmt.Fprintf(w, "  %s:\n", title)
			first = false
		}
		for _, value := range values {
			fmt.Fprintf(w, "    %s: %s\n", key, value)
		}
	}
}

func logRequest(req *Request) {
	var b = &bytes.Buffer{}
	fmt.Fprintf(b, "REQUEST\n")
	fmt.Fprintf(b, "  %s HTTP/%d.%d %s\n", req.Method, req.ProtocolVersion/1000, req.ProtocolVersion%1000, req.URL)
	fmt.Fprintf(b, "  RemoteAddr:  %s\n", req.RemoteAddr)
	fmt.Fprintf(b, "  ContentType:  %s\n", req.ContentType)
	fmt.Fprintf(b, "  ContentLength:  %d\n", req.ContentLength)
	writeStringMap(b, "Header", map[string][]string(req.Header))
	writeStringMap(b, "Param", map[string][]string(req.Param))
	writeStringMap(b, "Cookie", map[string][]string(req.Cookie))
	log.Print(b.String())
}

func logResponse(status int, header HeaderMap) {
	var b = &bytes.Buffer{}
	fmt.Fprintf(b, "RESPONSE\n")
	fmt.Fprintf(b, "  Status: %d\n", status)
	writeStringMap(b, "Header", header)
	log.Print(b.String())
}

// DebugLogger returns a handler that logs the request and response.
func DebugLogger(enabled bool, handler Handler) Handler {
	if !enabled {
		return handler
	}
	return HandlerFunc(func(req *Request) {
		logRequest(req)
		FilterRespond(req, func(status int, header HeaderMap) (int, HeaderMap) {
			logResponse(status, header)
			return status, header
		})
		handler.ServeWeb(req)
	})
}

// Logger returns a handler that logs the request
func Logger(log func(req *Request, status int, time int64), handler Handler) Handler {
	return HandlerFunc(func(req *Request) {
		var savedStatus int
		t := time.Nanoseconds()
		FilterRespond(req, func(status int, header HeaderMap) (int, HeaderMap) {
			savedStatus = status
			return status, header
		})
		handler.ServeWeb(req)
		log(req, savedStatus, time.Nanoseconds()-t)
	})
}
