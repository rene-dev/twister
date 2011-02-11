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

// The server package implements the HTTP protocol for a web server.
package server

import (
	"bufio"
	"bytes"
	"github.com/garyburd/twister/web"
	"http"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrBadRequestLine = os.NewError("could not parse request line")
)

// Server defines parameters for running an HTTP server.
type Server struct {
	// The server accepts incoming connections on this listener. The
	// application is required to set this field.
	Listener net.Listener

	// The server dispatches requests to this handler. The application is
	// required to set this field.
	Handler web.Handler

	// If true, then set the request URL protocol to HTTPS.
	Secure bool

	// Set request URL host to this string if host is not specified in the
	// request or headers.
	DefaultHost string

	// The net.Conn.SetReadTimeout value for new connections.
	ReadTimeout int64

	// The net.Conn.SetWriteTimeout value for new connections.
	WriteTimeout int64
}

type conn struct {
	server             *Server
	netConn            net.Conn
	br                 *bufio.Reader
	responseBody       web.ResponseBody
	chunked            bool
	closeAfterResponse bool
	hijacked           bool
	req                *web.Request
	requestAvail       int
	requestErr         os.Error
	respondCalled      bool
	responseAvail      int
	responseErr        os.Error
	write100Continue   bool
}

var requestLineRegexp = regexp.MustCompile("^([_A-Za-z0-9]+) ([^ ]+) HTTP/([0-9]+)\\.([0-9]+)[\r\n ]+$")

func readRequestLine(b *bufio.Reader) (method string, url string, version int, err os.Error) {

	p, err := b.ReadSlice('\n')
	if err != nil {
		if err == bufio.ErrBufferFull {
			err = web.ErrLineTooLong
		}
		return
	}

	m := requestLineRegexp.FindSubmatch(p)
	if m == nil {
		err = ErrBadRequestLine
		return
	}

	method = string(m[1])

	major, err := strconv.Atoi(string(m[3]))
	if err != nil {
		return
	}

	minor, err := strconv.Atoi(string(m[4]))
	if err != nil {
		return
	}

	version = web.ProtocolVersion(major, minor)

	url = string(m[2])

	return
}

func (c *conn) prepare() (err os.Error) {
	method, rawURL, version, err := readRequestLine(c.br)
	if err != nil {
		return err
	}

	header := web.HeaderMap{}
	err = header.ParseHttpHeader(c.br)
	if err != nil {
		return err
	}

	url, err := http.ParseURL(rawURL)
	if err != nil {
		return err
	}

	if url.Host == "" {
		url.Host = header.GetDef(web.HeaderHost, "")
		if url.Host == "" {
			url.Host = c.server.DefaultHost
		}
	}

	if c.server.Secure {
		url.Scheme = "https"
	} else {
		url.Scheme = "http"
	}

	req, err := web.NewRequest(c.netConn.RemoteAddr().String(), method, url, version, header)
	if err != nil {
		return
	}
	c.req = req

	c.requestAvail = req.ContentLength
	if c.requestAvail < 0 {
		c.requestAvail = 0
	}

	if s, found := req.Header.Get(web.HeaderExpect); found {
		c.write100Continue = strings.ToLower(s) == "100-continue"
	}

	connection := strings.ToLower(req.Header.GetDef(web.HeaderConnection, ""))
	if version >= web.ProtocolVersion(1, 1) {
		c.closeAfterResponse = connection == "close"
	} else if version == web.ProtocolVersion(1, 0) && req.ContentLength >= 0 {
		c.closeAfterResponse = connection != "keep-alive"
	} else {
		c.closeAfterResponse = true
	}

	req.Responder = c
	req.Body = requestReader{c}
	return nil
}

type requestReader struct {
	*conn
}

func (c requestReader) Read(p []byte) (int, os.Error) {
	if c.requestErr != nil {
		return 0, c.requestErr
	}
	if c.write100Continue {
		c.write100Continue = false
		io.WriteString(c.netConn, "HTTP/1.1 100 Continue\r\n\r\n")
	}
	if c.requestAvail <= 0 {
		c.requestErr = os.EOF
		return 0, c.requestErr
	}
	if len(p) > c.requestAvail {
		p = p[0:c.requestAvail]
	}
	var n int
	n, c.requestErr = c.br.Read(p)
	c.requestAvail -= n
	return n, c.requestErr
}

func (c *conn) Respond(status int, header web.HeaderMap) (body web.ResponseBody) {
	if c.hijacked {
		log.Println("twister: Respond called on hijacked connection")
		return nil
	}
	if c.respondCalled {
		log.Println("twister: multiple calls to Respond")
		return nil
	}
	c.respondCalled = true
	c.requestErr = web.ErrInvalidState

	if _, found := header.Get(web.HeaderTransferEncoding); found {
		log.Println("twister: transfer encoding not allowed")
		header[web.HeaderTransferEncoding] = nil, false
	}

	if c.requestAvail > 0 {
		c.closeAfterResponse = true
	}

	c.chunked = true
	c.responseAvail = 0

	if status == web.StatusNotModified {
		header[web.HeaderContentType] = nil, false
		header[web.HeaderContentLength] = nil, false
		c.chunked = false
	} else if s, found := header.Get(web.HeaderContentLength); found {
		c.responseAvail, _ = strconv.Atoi(s)
		c.chunked = false
	} else if c.req.ProtocolVersion < web.ProtocolVersion(1, 1) {
		c.closeAfterResponse = true
	}

	if c.closeAfterResponse {
		header.Set(web.HeaderConnection, "close")
		c.chunked = false
	}

	if c.req.Method == "HEAD" {
		c.chunked = false
	}

	if c.chunked {
		header.Set(web.HeaderTransferEncoding, "chunked")
	}

	proto := "HTTP/1.0"
	if c.req.ProtocolVersion >= web.ProtocolVersion(1, 1) {
		proto = "HTTP/1.1"
	}
	statusString := strconv.Itoa(status)
	text := web.StatusText(status)

	var b bytes.Buffer
	b.WriteString(proto)
	b.WriteString(" ")
	b.WriteString(statusString)
	b.WriteString(" ")
	b.WriteString(text)
	b.WriteString("\r\n")
	header.WriteHttpHeader(&b)

	if c.req.Method == "HEAD" {
		c.responseBody = nullResponseBody{c}
		_, c.responseErr = c.netConn.Write(b.Bytes())
	} else if c.chunked {
		c.responseBody = bufio.NewWriter(chunkedWriter{c})
		_, c.responseErr = c.netConn.Write(b.Bytes())
	} else {
		c.responseBody = bufio.NewWriter(identityWriter{c})
		c.responseBody.Write(b.Bytes())
	}

	return c.responseBody
}

func (c *conn) Hijack() (conn net.Conn, buf []byte, err os.Error) {
	if c.respondCalled {
		return nil, nil, web.ErrInvalidState
	}

	conn = c.netConn
	buf, err = c.br.Peek(c.br.Buffered())
	if err != nil {
		panic("twisted.server: unexpected error peeking at bufio")
	}

	c.hijacked = true
	c.requestErr = web.ErrInvalidState
	c.responseErr = web.ErrInvalidState
	c.req = nil
	c.br = nil
	c.netConn = nil

	return
}

// Finish the HTTP request
func (c *conn) finish() os.Error {
	if !c.respondCalled {
		c.req.Respond(web.StatusOK, web.HeaderContentType, "text/html charset=utf-8")
	}
	if c.responseAvail != 0 {
		c.closeAfterResponse = true
	}
	c.responseBody.Flush()
	if c.chunked {
		_, c.responseErr = io.WriteString(c.netConn, "0\r\n\r\n")
	}
	if c.responseErr == nil {
		c.responseErr = web.ErrInvalidState
	}
	c.netConn = nil
	c.br = nil
	c.responseBody = nil
	return nil
}

type nullResponseBody struct {
	*conn
}

func (c nullResponseBody) Write(p []byte) (int, os.Error) {
	if c.responseErr != nil {
		return 0, c.responseErr
	}
	return len(p), nil
}

func (c nullResponseBody) Flush() os.Error {
	return c.responseErr
}

type identityWriter struct {
	*conn
}

func (c identityWriter) Write(p []byte) (int, os.Error) {
	if c.responseErr != nil {
		return 0, c.responseErr
	}
	var n int
	n, c.responseErr = c.netConn.Write(p)
	c.responseAvail -= n
	return n, c.responseErr
}

type chunkedWriter struct {
	*conn
}

func (c chunkedWriter) Write(p []byte) (int, os.Error) {
	if c.responseErr != nil {
		return 0, c.responseErr
	}
	if len(p) == 0 {
		return 0, nil
	}
	_, c.responseErr = io.WriteString(c.netConn, strconv.Itob(len(p), 16)+"\r\n")
	if c.responseErr != nil {
		return 0, c.responseErr
	}
	var n int
	n, c.responseErr = c.netConn.Write(p)
	if c.responseErr != nil {
		return n, c.responseErr
	}
	_, c.responseErr = io.WriteString(c.netConn, "\r\n")
	return n, c.responseErr
}

func (s *Server) serveConnection(netConn net.Conn) {
	if s.ReadTimeout != 0 {
		netConn.SetReadTimeout(s.ReadTimeout)
	}
	if s.WriteTimeout != 0 {
		netConn.SetWriteTimeout(s.WriteTimeout)
	}
	br := bufio.NewReader(netConn)
	for {
		c := conn{
			server:  s,
			netConn: netConn,
			br:      br}
		if err := c.prepare(); err != nil {
			if err != os.EOF {
				log.Println("twister/server: prepare failed", err)
			}
			break
		}
		s.Handler.ServeWeb(c.req)
		if c.hijacked {
			return
		}
		if err := c.finish(); err != nil {
			log.Println("twister/server: finish failed", err)
			break
		}
		if c.closeAfterResponse {
			break
		}
	}
	netConn.Close()
}

// Serve accepts incoming HTTP connections on s.Listener, creating a new
// goroutine for each. The goroutines read requests and then call s.Handler to
// respond to the request.
//
// The "Hello World" server using Serve() is:
//
//  package main
//  
//  import (
//      "github.com/garyburd/twister/web"
//      "github.com/garyburd/twister/server"
//      "io"
//      "log"
//      "net"
//  )
//
//  func helloHandler(req *web.Request) {
//      w := req.Respond(web.StatusOK, web.HeaderContentType, "text/plain")
//      io.WriteString(w, "Hello, World!\n")
//  }
//  
//  func main() {
//      handler := web.NewRouter().Register("/", "GET", helloHandler)
//      listener, err := net.Listen("tcp", ":8080")
//      if err != nil {
//          log.Fatal("Listen", err)
//      }
//      defer listener.Close()
//      err = (&server.Server{Listener: listener, Handler: handler}).Serve()
//      if err != nil {
//          log.Fatal("Server", err)
//      }
//  }
func (s *Server) Serve() os.Error {
	for {
		netConn, e := s.Listener.Accept()
		if e != nil {
			return e
		}
		go s.serveConnection(netConn)
	}
	return nil
}

// Run is a convenience function for running an HTTP server. Run listens on the
// TCP address addr, initializes a server object and calls the server's Serve()
// method to handle HTTP requests. Run logs a fatal error if it encounters an
// error.
//
// The Server object is initialized with the handler argument and listener. If
// the application needs to set any other Server fields or if the application
// needs to create the listener, then the application should directly create
// the Server object and call the Serve() method.
//
// The "Hello World" server using Run() is:
//
//  package main
//  
//  import (
//      "github.com/garyburd/twister/web"
//      "github.com/garyburd/twister/server"
//      "io"
//  )
//  
//  func helloHandler(req *web.Request) {
//      w := req.Respond(web.StatusOK, web.HeaderContentType, "text/plain")
//      io.WriteString(w, "Hello, World!\n")
//  }
//  
//  func main() {
//      server.Run(":8080", web.NewRouter().Register("/", "GET", helloHandler))
//  }
//
func Run(addr string, handler web.Handler) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("Listen", err)
		return
	}
	defer listener.Close()
	err = (&Server{Listener: listener, Handler: handler}).Serve()
	if err != nil {
		log.Fatal("Server", err)
	}
}
