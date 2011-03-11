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

package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"github.com/garyburd/twister/web"
	"os"
)

// LogRecord records information about a request for logging.
type LogRecord struct {
	// The request, possibly modified by handlers.
	Request *web.Request

	// Errors encoutered while handling request. 
	Error os.Error

	// Response status.
	Status int

	// Response headers.
	Header web.HeaderMap

	// Number of bytes written to output including headers and transfer encoding.
	Written int
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

// ShortLogger logs a short summary of the request.
func ShortLogger(lr *LogRecord) {
	if lr.Error != nil {
		log.Printf("%d %s %s %s\n", lr.Status, lr.Request.Method, lr.Request.URL, lr.Error)
	} else {
		log.Printf("%d %s %s\n", lr.Status, lr.Request.Method, lr.Request.URL)
	}
}

// VerboseLogger prints out just about everything about the request and response.
func VerboseLogger(lr *LogRecord) {
	var b = &bytes.Buffer{}
	fmt.Fprintf(b, "REQUEST\n")
	fmt.Fprintf(b, "  %s HTTP/%d.%d %s\n", lr.Request.Method, lr.Request.ProtocolVersion/1000, lr.Request.ProtocolVersion%1000, lr.Request.URL)
	fmt.Fprintf(b, "  RemoteAddr:  %s\n", lr.Request.RemoteAddr)
	fmt.Fprintf(b, "  ContentType:  %s\n", lr.Request.ContentType)
	fmt.Fprintf(b, "  ContentLength:  %d\n", lr.Request.ContentLength)
	writeStringMap(b, "Header", map[string][]string(lr.Request.Header))
	writeStringMap(b, "Param", map[string][]string(lr.Request.Param))
	writeStringMap(b, "Cookie", map[string][]string(lr.Request.Cookie))
	fmt.Fprintf(b, "RESPONSE\n")
	fmt.Fprintf(b, "  Error: %v\n", lr.Error)
	fmt.Fprintf(b, "  Status: %d\n", lr.Status)
	fmt.Fprintf(b, "  Written: %d\n", lr.Written)
	writeStringMap(b, "Header", lr.Header)
	log.Print(b.String())
}
