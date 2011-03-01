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
)

type ParseUrlEncodedFormTest struct {
	s string
	m ParamMap
}

var ParseUrlEncodedFormTests = []ParseUrlEncodedFormTest{
	{"a=", ParamMap{"a": []string{""}}},
	{"a=b", ParamMap{"a": []string{"b"}}},
	{"a=b&c=d", ParamMap{"a": []string{"b"}, "c": []string{"d"}}},
	{"a=b&a=c", ParamMap{"a": []string{"b", "c"}}},
	{"a=Hello%20World", ParamMap{"a": []string{"Hello World"}}},
}

func TestParseUrlEncodedForm(t *testing.T) {
	for _, pt := range ParseUrlEncodedFormTests {
		p := []byte(pt.s)
		m := make(ParamMap)
		m.ParseFormEncodedBytes(p)
		if !reflect.DeepEqual(pt.m, m) {
			t.Errorf("ParseFormEncodedBytes(%q) = %q, want %q", pt.s, m, pt.m)
		}
	}
}
