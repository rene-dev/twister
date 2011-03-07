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
	"io"
	"os"
	"github.com/garyburd/twister/web"
	"bufio"
)

type responseData struct {
	// Number of bytes written to output including headers and transfer encoding.
	written int

	// Number of header bytes written.
	headerWritten int
}

type responseBody interface {
	web.ResponseBody

	// finish the response body and return an error if the connection should be
	// closed due to a write error.
	finish() (responseData, os.Error)
}

// nullResponseBody discoards the response body.
type nullResponseBody struct {
	err          os.Error
	responseData responseData
}

func newNullResponseBody(wr io.Writer, header []byte) (*nullResponseBody, os.Error) {
	w := &nullResponseBody{}
	w.responseData.headerWritten, w.err = wr.Write(header)
	w.responseData.written = w.responseData.headerWritten
	return w, w.err
}

func (w *nullResponseBody) Write(p []byte) (int, os.Error) {
	if w.err != nil {
		return 0, w.err
	}
	return len(p), nil
}

func (w *nullResponseBody) Flush() os.Error {
	return w.err
}

func (w *nullResponseBody) finish() (responseData, os.Error) {
	err := w.err
	if w.err == nil {
		w.err = web.ErrInvalidState
	}
	return w.responseData, err
}

// identityResponseBody implements identity encoding of the response body. 
type identityResponseBody struct {
	err os.Error
	bw  *bufio.Writer

	// Value of Content-Length header.
	contentLength int

	// Bytes written.
	responseData responseData
}

func newIdentityResponseBody(wr io.Writer, header []byte, bufferSize, contentLength int) (*identityResponseBody, os.Error) {
	w := &identityResponseBody{contentLength: contentLength}

	w.bw, w.err = bufio.NewWriterSize(wr, bufferSize)
	if w.err != nil {
		return w, w.err
	}

	w.responseData.headerWritten, w.err = w.bw.Write(header)
	w.responseData.written = w.responseData.headerWritten
	return w, w.err
}

func (w *identityResponseBody) Write(p []byte) (int, os.Error) {
	if w.err != nil {
		return 0, w.err
	}
	var n int
	n, w.err = w.bw.Write(p)
	w.responseData.written += n
	if w.err == nil && w.contentLength >= 0 && (w.responseData.written - w.responseData.headerWritten) > w.contentLength {
		w.err = os.NewError("twister: long write by handler")
	}
	return n, w.err
}

func (w *identityResponseBody) Flush() os.Error {
	if w.err != nil {
		return w.err
	}
	w.err = w.bw.Flush()
	return w.err
}

func (w *identityResponseBody) finish() (responseData, os.Error) {
	w.Flush()
	if w.err != nil {
		return w.responseData, w.err
	}
	if w.contentLength >= 0 && (w.responseData.written - w.responseData.headerWritten) < w.contentLength {
		w.err = os.NewError("twister: short write by handler")
	}
	err := w.err
	if w.err == nil {
		w.err = web.ErrInvalidState
	}
	return w.responseData, err
}

type chunkedResponseBody struct {
	err     os.Error  // error from wr
	wr      io.Writer // write here
	buf     []byte    // buffered output
	s       int       // start of chunk in buf 
	n       int       // current write position in buf
	ndigit  int       // number of hex digits in chunk size
	responseData responseData // Bytes written
}

func newChunkedResponseBody(wr io.Writer, header []byte, bufferSize int) (*chunkedResponseBody, os.Error) {
	w := &chunkedResponseBody{wr: wr, buf: make([]byte, bufferSize)}

	for n := int32(bufferSize); n != 0; n >>= 4 {
		w.ndigit += 1
	}

	if len(header) < len(w.buf) {
		w.n = copy(w.buf, header)
	} else {
		w.responseData.headerWritten, w.err = w.wr.Write(header)
		w.responseData.written = w.responseData.headerWritten
	}

	w.s = w.n
	w.n += w.ndigit + 2
	return w, w.err
}

func (w *chunkedResponseBody) writeBuf() {
	var n int
	n, w.err = w.wr.Write(w.buf[:w.n])
	w.responseData.written += n
}

func (w *chunkedResponseBody) finalizeChunk() {
	if w.s+w.ndigit+2 == w.n {
		// The chunk is empty. Reset back start of chunk.
		w.n = w.s
		return
	}

	n := w.n - w.s - w.ndigit - 2

	// CRLF after data.
	w.buf[w.n] = '\r'
	w.buf[w.n+1] = '\n'
	w.n += 2

	// CRLF before data.
	w.buf[w.s+w.ndigit] = '\r'
	w.buf[w.s+w.ndigit+1] = '\n'

	// Length with 0 padding
	for i := w.s + w.ndigit - 1; i >= w.s; i-- {
		w.buf[i] = "0123456789abcdef"[n&0xf]
		n = n >> 4
	}
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *chunkedResponseBody) Flush() os.Error {
	if w.err != nil {
		return w.err
	}
	w.finalizeChunk()
	if w.n > 0 {
		w.writeBuf()
		if w.err != nil {
			return w.err
		}
	}
	w.s = 0
	w.n = w.ndigit + 2 // length CRLF
	return nil
}

func (w *chunkedResponseBody) finish() (responseData, os.Error) {
	if w.err != nil {
		return w.responseData, w.err
	}
	w.finalizeChunk()
	const last = "0\r\n\r\n"
	if w.n+len(last) > len(w.buf) {
		w.writeBuf()
		if w.err != nil {
			return w.responseData, w.err
		}
		w.n = 0
	}
	copy(w.buf[w.n:], last)
	w.n += len(last)
	w.writeBuf()
	err := w.err
	if w.err == nil {
		w.err = web.ErrInvalidState
	}
	return w.responseData, err
}

func (w *chunkedResponseBody) Write(p []byte) (int, os.Error) {
	if w.err != nil {
		return 0, w.err
	}
	nn := 0
	for len(p) > 0 {
		n := len(w.buf) - w.n - 2 // 2 for CRLF after data
		if n <= 0 {
			w.Flush()
			if w.err != nil {
				break
			}
			n = len(w.buf) - w.n - 2 // 2 for CRLF after data
		}
		if n > len(p) {
			n = len(p)
		}
		copy(w.buf[w.n:], p)
		w.n += n
		nn += n
		p = p[n:]
	}
	return nn, w.err
}
