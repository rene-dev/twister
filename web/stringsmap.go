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

// The web package defines the application programming interface to a web
// server and implements functionality common to many web applications.
package web

import (
	"bytes"
	"http"
)

// StringsMap maps strings to slices of strings.
type StringsMap map[string][]string

// NewStringsMap returns a map initialized with the given key-value pairs.
func NewStringsMap(kvs ...string) StringsMap {
	if len(kvs)%2 == 1 {
		panic("twister: even number args required for NewStringsMap")
	}
	m := make(StringsMap)
	for i := 0; i < len(kvs); i += 2 {
		m.Append(kvs[i], kvs[i+1])
	}
	return m
}

// Get returns the first value for given key or "" if the key is not found.
func (m StringsMap) Get(key string) (value string, found bool) {
	values, found := m[key]
	if !found || len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetDef returns first value for given key, or def if the key is not found.
func (m StringsMap) GetDef(key string, def string) string {
	values, found := m[key]
	if !found || len(values) == 0 {
		return def
	}
	return values[0]
}

// Append value to slice for given key.
func (m StringsMap) Append(key string, value string) {
	m[key] = append(m[key], value)
}

// Set value for given key, discarding previous values if any.
func (m StringsMap) Set(key string, value string) {
	m[key] = []string{value}
}

// StringMap returns a string to string map by discarding all but the first
// value for a key.
func (m StringsMap) StringMap() map[string]string {
	result := make(map[string]string)
	for key, values := range m {
		result[key] = values[0]
	}
	return result
}

// FormEncode returns a buffer containing the URL form encoding of the map.
func (m StringsMap) FormEncode() []byte {
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

// FormEncode returns a string containing the URL form encoding of the map.
func (m StringsMap) FormEncodeString() string {
	return string(m.FormEncode())
}
