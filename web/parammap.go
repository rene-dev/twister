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
	"http"
	"os"
)

// ParamMap maps strings to slices of strings.
type ParamMap map[string][]string

// NewParamMap returns a map initialized with the given key-value pairs.
func NewParamMap(kvs ...string) ParamMap {
	if len(kvs)%2 == 1 {
		panic("twister: even number args required for NewParamMap")
	}
	m := make(ParamMap)
	for i := 0; i < len(kvs); i += 2 {
		m.Add(kvs[i], kvs[i+1])
	}
	return m
}

// Get returns the first value for given key or "" if the key is not found.
func (m ParamMap) Get(key string) string {
	values, found := m[key]
	if !found || len(values) == 0 {
		return ""
	}
	return values[0]
}

// Add appends value to slice for given key.
func (m ParamMap) Add(key string, value string) {
	m[key] = append(m[key], value)
}

// Set value for given key, discarding previous values if any.
func (m ParamMap) Set(key string, value string) {
	m[key] = []string{value}
}

// StringMap returns a string to string map by discarding all but the first
// value for a key.
func (m ParamMap) StringMap() map[string]string {
	result := make(map[string]string)
	for key, values := range m {
		result[key] = values[0]
	}
	return result
}

// FormEncodedBytes returns a buffer containing the URL form encoding of the
// map.
func (m ParamMap) FormEncodedBytes() []byte {
	var b bytes.Buffer
	sep := false
	for key, values := range m {
		escapedKey := http.URLEscape(key)
		for _, value := range values {
			if sep {
				b.WriteByte('&')
			} else {
				sep = true
			}
			b.WriteString(escapedKey)
			b.WriteByte('=')
			b.WriteString(http.URLEscape(value))
		}
	}
	return b.Bytes()
}

// FormEncodedString returns a string containing the URL form encoding of the
// map.
func (m ParamMap) FormEncodedString() string {
	return string(m.FormEncodedBytes())
}

const notHex = 127

func dehex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return notHex
}

// ParseFormEncodedBytes parses the URL-encoded form and appends the values to
// the supplied map. This function modifies the contents of p.
func (m ParamMap) ParseFormEncodedBytes(p []byte) os.Error {
	key := ""
	j := 0
	for i := 0; i < len(p); {
		switch p[i] {
		case '=':
			key = string(p[0:j])
			j = 0
			i += 1
		case '&':
			m.Add(key, string(p[0:j]))
			key = ""
			j = 0
			i += 1
		case '%':
			if i+2 >= len(p) {
				return ErrBadFormat
			}
			a := dehex(p[i+1])
			b := dehex(p[i+2])
			if a == notHex || b == notHex {
				return ErrBadFormat
			}
			p[j] = a<<4 | b
			j += 1
			i += 3
		case '+':
			p[j] = ' '
			j += 1
			i += 1
		default:
			p[j] = p[i]
			j += 1
			i += 1
		}
	}
	if key != "" {
		m.Add(key, string(p[0:j]))
	}
	return nil
}
