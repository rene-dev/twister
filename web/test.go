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
	"os"
	"http"
	"bytes"
	"net"
)

type testResponseBody struct {
	bytes.Buffer
}

type testResponder struct {
	body   testResponseBody
	status int
	header StringsMap
}

func (r *testResponder) Respond(status int, header StringsMap) ResponseBody {
	r.status = status
	r.header = header
	return &r.body
}

func (r *testResponder) Hijack() (net.Conn, []byte, os.Error) {
	return nil, nil, os.NewError("Not supported")
}

func (b *testResponseBody) Flush() os.Error {
	return nil
}

// RunHandler runs the a handler with a request created from the arguments and
// returns the respponse. This function is intended to be used in tests.
func RunHandler(url string, method string, reqHeader StringsMap, reqBody []byte, handler Handler) (status int, header StringsMap, respBody []byte) {
	remoteAddr := "1.2.3.4"
	protocolVersion := ProtocolVersion11
	if reqHeader == nil {
		reqHeader = make(StringsMap)
	}
	parsedURL, err := http.ParseURL(url)
	if err != nil {
		panic(err)
	}
	req, err := NewRequest(remoteAddr, method, parsedURL, protocolVersion, reqHeader)
	if err != nil {
		panic(err)
	}
	var resp testResponder
	req.Body = bytes.NewBuffer(reqBody)
	req.Responder = &resp
	handler.ServeWeb(req)
	return resp.status, resp.header, resp.body.Bytes()
}
