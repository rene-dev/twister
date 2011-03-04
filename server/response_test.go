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
	"testing"
	"regexp"
)

var chunkHeaderRegexp = regexp.MustCompile("^[0-9A-Z]+\r\n")

var dots = string(bytes.Repeat([]byte{'.'}, 4096))

const chunkTestBufferSize = 32

var chunkedResponseTests = []struct {
	n   []int
	out string
}{
	// Header only
	{[]int{10}, dots[:10] + "0\r\n\r\n"},
	// Header one byte smaller than buffer size
	{[]int{31}, dots[:31] + "0\r\n\r\n"},
	// Header size = buffer size
	{[]int{32}, dots[:32] + "0\r\n\r\n"},
	// Chunk fits in buffer
	{[]int{0, 26}, "1a\r\n" + dots[:26] + "\r\n0\r\n\r\n"},
	// Chunk one byte larger than buffer
	{[]int{0, 27}, "1a\r\n" + dots[:26] + "\r\n01\r\n" + "." + "\r\n0\r\n\r\n"},
	// Flush before and after chunk
	{[]int{10, -1, 10, -1}, dots[:10] + "0a\r\n" + dots[:10] + "\r\n0\r\n\r\n"},
	// Chunk in multipe writes
	{[]int{10, -1, 5, 5, -1}, dots[:10] + "0a\r\n" + dots[:10] + "\r\n0\r\n\r\n"},
	{[]int{10, -1, 5, -1, 5, -1}, dots[:10] + "05\r\n" + dots[:5] + "\r\n05\r\n" + dots[:5] + "\r\n0\r\n\r\n"},
}

func TestChunkedResponse(t *testing.T) {
	for _, tt := range chunkedResponseTests {
		var buf bytes.Buffer
		nn := tt.n[0]
		w, _ := newChunkedResponseBody(&buf, []byte(dots[:nn]), chunkTestBufferSize)
		for i := 1; i < len(tt.n); i++ {
			n := tt.n[i]
			if n < 0 {
				w.Flush()
			} else {
				w.Write([]byte(dots[:n]))
				nn += n
			}
		}
		n, _ := w.finish()
		if n != len(tt.out) {
			t.Errorf("%v, written = %d, want %d", tt.n, n, len(tt.out))
		}
		out := buf.String()
		if out != tt.out {
			t.Errorf("%v\ngot:  %q\nwant: %q", tt.n, out, tt.out)
		}
	}
}
