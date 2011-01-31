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
	"testing"
	"reflect"
	"bufio"
	"bytes"
)

type ParseHttpHeaderTest struct {
	name   string
	header HeaderMap
	s      string
}

var ParseHttpHeaderTests = []ParseHttpHeaderTest{
	{"multihdr", NewHeaderMap(
		HeaderContentType, "text/html",
		HeaderCookie, "hello=world",
		HeaderCookie, "foo=bar"),
		`Content-Type: text/html
CoOkie: hello=world
Cookie: foo=bar

`},
	{"continuation", NewHeaderMap(
		HeaderContentType, "text/html",
		HeaderCookie, "hello=world, foo=bar"),
		`Cookie: hello=world,
 foo=bar
Content-Type: text/html

`},
}

func TestParseHttpHeader(t *testing.T) {
	for _, tt := range ParseHttpHeaderTests {
		b := bufio.NewReader(bytes.NewBufferString(tt.s))
		header := HeaderMap{}
		err := header.ParseHttpHeader(b)
		if err != nil {
			t.Errorf("%s: expected error", tt.name)
		}
		if !reflect.DeepEqual(tt.header, header) {
			t.Errorf("%s bad header\nexpected: %q\nactual:   %q", tt.name, tt.header, header)
		}
	}
}
