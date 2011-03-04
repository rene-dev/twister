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

package server

import (
	"bytes"
	"github.com/garyburd/twister/web"
	"net"
	"os"
	"testing"
)

type testAddr string

func (a testAddr) Network() string {
	return string(a)
}

func (a testAddr) String() string {
	return string(a)
}

type testListener struct {
	in, out     bytes.Buffer
	done        chan bool
	acceptCount int
}

func (l *testListener) Accept() (conn net.Conn, err os.Error) {
	if l.acceptCount > 0 {
		return nil, os.EOF
	}
	l.acceptCount += 1
	l.done = make(chan bool)
	return testConn{l}, nil
}

func (l *testListener) Close() os.Error {
	return nil
}

func (l *testListener) Addr() net.Addr {
	return testAddr("listen")
}

type testConn struct {
	*testListener
}

func (c testConn) Read(b []byte) (int, os.Error) {
	return c.in.Read(b)
}

func (c testConn) Write(b []byte) (int, os.Error) {
	return c.out.Write(b)
}

func (c testConn) Close() os.Error {
	c.done <- true
	return nil
}

func (c testConn) LocalAddr() net.Addr {
	return testAddr("local")
}

func (c testConn) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (c testConn) SetTimeout(nsec int64) os.Error {
	return nil
}

func (c testConn) SetReadTimeout(nsec int64) os.Error {
	return nil
}

func (c testConn) SetWriteTimeout(nsec int64) os.Error {
	return nil
}

func testHandler(req *web.Request) {
	req.ParseForm(1000)
	header := make(web.HeaderMap)
	if s := req.Param.Get("cl"); s != "" {
		header.Set(web.HeaderContentLength, s)
	}
	w := req.Responder.Respond(web.StatusOK, header)
	if s := req.Param.Get("w"); s != "" {
		w.Write([]byte(s))
	}
}

var serverTests = []struct {
	in  string
	out string
}{
	{
		"GET /?w=Hello HTTP/1.0\r\n\r\n",
		"HTTP/1.0 200 OK\r\nConnection: close\r\n\r\nHello",
	},
	{
		"GET /?w=Hello HTTP/1.0\r\nConnection: keep-alive\r\n\r\n",
		"HTTP/1.0 200 OK\r\nConnection: close\r\n\r\nHello",
	},
	{
		"GET /?cl=5&w=Hello HTTP/1.0\r\n\r\n",
		"HTTP/1.0 200 OK\r\nConnection: close\r\nContent-Length: 5\r\n\r\nHello",
	},
	{
		"GET /?cl=5&w=Hello HTTP/1.0\r\nConnection: keep-alive\r\n\r\n",
		"HTTP/1.0 200 OK\r\nContent-Length: 5\r\n\r\nHello",
	},
	{
		"GET /?w=Hello HTTP/1.1\r\n\r\n",
		"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n0005\r\nHello\r\n0\r\n\r\n",
	},
	{
		"GET /?cl=5&w=Hello HTTP/1.1\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello",
	},
	{
		// POST
		"POST /?cl=5 HTTP/1.1\r\nContent-Length: 7\r\nContent-Type: application/x-www-form-urlencoded\r\n\r\nw=Hello",
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello",
	},
	{
		// POST with expect
		"POST /?cl=5 HTTP/1.1\r\nContent-Length: 7\r\nContent-Type: application/x-www-form-urlencoded\r\nExpect: 100-continue\r\n\r\nw=Hello",
		"HTTP/1.1 100 Continue\r\n\r\nHTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello",
	},
	{
		// Expect connection close because request body not read by handler.
		"POST /?cl=0 HTTP/1.1\r\nContent-Length: 7\r\n\r\nw=Hello",
		"HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Length: 0\r\n\r\n",
	},
	{
		// Two requests with identity encoded resposne.
		"GET /?cl=5&w=Hello HTTP/1.1\r\n\r\n" +
			"GET /?cl=5&w=Hello HTTP/1.1\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello" +
			"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello",
	},
	{
		// Two requests with chunked encoded response.
		"GET /?w=Hello HTTP/1.1\r\n\r\n" +
			"GET /?w=Hello HTTP/1.1\r\n\r\n",
		"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n0005\r\nHello\r\n0\r\n\r\n" +
			"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n0005\r\nHello\r\n0\r\n\r\n",
	},
	{
		// HEAD
		"HEAD /?cl=5&w=Hello HTTP/1.1\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\n",
	},
}

func TestServer(t *testing.T) {
	for _, st := range serverTests {
		l := &testListener{}
		l.in.WriteString(st.in)
		err := (&Server{Listener: l, Handler: web.HandlerFunc(testHandler)}).Serve()
		if err != os.EOF {
			t.Errorf("Server() = %v", err)
		}
		<-l.done
		out := l.out.String()
		if out != st.out {
			t.Errorf("in=%q\ngot:  %q\nwant: %q", st.in, out, st.out)
		}
	}
}
