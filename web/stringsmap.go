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
	"io"
	"bufio"
)

var (
	ErrLineTooLong    = os.NewError("HTTP header line too long")
	ErrBadHeaderLine  = os.NewError("could not parse HTTP header line")
	ErrHeaderTooLong  = os.NewError("HTTP header value too long")
	ErrHeadersTooLong = os.NewError("too many HTTP headers")
)

// StringsMap maps strings to slices of strings. StringsMaps are used to
// represent HTTP request parameters and HTTP headers.
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

// FormEncoding returns a buffer containing the URL form encoding of the map.
func (m StringsMap) FormEncoding() []byte {
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

// FormEncoding returns a string containing the URL form encoding of the map.
func (m StringsMap) FormEncodingString() string {
	return string(m.FormEncoding())
}

// WriteHttpHeader writes the map in HTTP header format.
func (m StringsMap) WriteHttpHeader(w io.Writer) os.Error {
	for key, values := range m {
		keyBytes := []byte(key)
		for _, value := range values {
			if _, err := w.Write(keyBytes); err != nil {
				return err
			}
			if _, err := w.Write(colonSpaceBytes); err != nil {
				return err
			}
			valueBytes := []byte(value)
			// convert \r and \n to space to prevent response splitting attacks.
			for i, c := range valueBytes {
				if c == '\r' || c == '\n' {
					valueBytes[i] = ' '
				}
			}
			if _, err := w.Write(valueBytes); err != nil {
				return err
			}
			if _, err := w.Write(crlfBytes); err != nil {
				return err
			}
		}
	}
	_, err := w.Write(crlfBytes)
	return err
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

// ParseUrlEncodedFormBytes parses the URL-encoded form and appends the values to
// the supplied map. This function modifies the contents of p.
func (m StringsMap) ParseUrlEncodedFormBytes(p []byte) os.Error {
	key := ""
	j := 0
	for i := 0; i < len(p); {
		switch p[i] {
		case '=':
			key = string(p[0:j])
			j = 0
			i += 1
		case '&':
			m.Append(key, string(p[0:j]))
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
		m.Append(key, string(p[0:j]))
	}
	return nil
}

// ParseHttpHeader parses the HTTP headers and appends the values to the
// supplied map. Header names are converted to canonical format.
func (m StringsMap) ParseHttpHeader(b *bufio.Reader) (err os.Error) {

	const (
		// Max size for header line
		maxLineSize = 4096
		// Max size for header value
		maxValueSize = 4096
		// Maximum number of headers 
		maxHeaderCount = 256
	)

	lastKey := ""
	headerCount := 0

	for {
		p, err := b.ReadSlice('\n')
		if err != nil {
			if err == bufio.ErrBufferFull {
				err = ErrLineTooLong
			} else if err == os.EOF {
				err = io.ErrUnexpectedEOF
			}
			return err
		}

		// remove line terminator
		if len(p) >= 2 && p[len(p)-2] == '\r' {
			// \r\n
			p = p[0 : len(p)-2]
		} else {
			// \n
			p = p[0 : len(p)-1]
		}

		// End of headers?
		if len(p) == 0 {
			break
		}

		// Don't allow huge header lines.
		if len(p) > maxLineSize {
			return ErrLineTooLong
		}

		if IsSpaceByte(p[0]) {

			if lastKey == "" {
				return ErrBadHeaderLine
			}

			p = trimWSLeft(trimWSRight(p))

			if len(p) > 0 {
				values := m[lastKey]
				value := values[len(values)-1]
				value = value + " " + string(p)
				if len(value) > maxValueSize {
					return ErrHeaderTooLong
				}
				values[len(values)-1] = value
			}

		} else {

			// New header
			headerCount = headerCount + 1
			if headerCount > maxHeaderCount {
				return ErrHeadersTooLong
			}

			// Key
			i := skipBytes(p, IsTokenByte)
			if i < 1 {
				return ErrBadHeaderLine
			}
			key := HeaderNameBytes(p[0:i])
			p = p[i:]
			lastKey = key

			p = trimWSLeft(p)

			// Colon
			if p[0] != ':' {
				return ErrBadHeaderLine
			}
			p = p[1:]

			// Value 
			p = trimWSLeft(p)
			value := string(trimWSRight(p))
			m.Append(key, value)
		}
	}
	return nil
}

func skipBytes(p []byte, f func(byte) bool) int {
	i := 0
	for ; i < len(p); i++ {
		if !f(byte(p[i])) {
			break
		}
	}
	return i
}

func trimWSLeft(p []byte) []byte {
	return p[skipBytes(p, IsSpaceByte):]
}

func trimWSRight(p []byte) []byte {
	var i int
	for i = len(p); i > 0; i-- {
		if !IsSpaceByte(p[i-1]) {
			break
		}
	}
	return p[0:i]
}

// Canonical header name constants.
const (
	HeaderAccept               = "Accept"
	HeaderAcceptCharset        = "Accept-Charset"
	HeaderAcceptEncoding       = "Accept-Encoding"
	HeaderAcceptLanguage       = "Accept-Language"
	HeaderAcceptRanges         = "Accept-Ranges"
	HeaderAge                  = "Age"
	HeaderAllow                = "Allow"
	HeaderAuthorization        = "Authorization"
	HeaderCacheControl         = "Cache-Control"
	HeaderConnection           = "Connection"
	HeaderContentEncoding      = "Content-Encoding"
	HeaderContentLanguage      = "Content-Language"
	HeaderContentLength        = "Content-Length"
	HeaderContentLocation      = "Content-Location"
	HeaderContentMD5           = "Content-Md5"
	HeaderContentRange         = "Content-Range"
	HeaderContentType          = "Content-Type"
	HeaderCookie               = "Cookie"
	HeaderDate                 = "Date"
	HeaderETag                 = "Etag"
	HeaderEtag                 = "Etag"
	HeaderExpect               = "Expect"
	HeaderExpires              = "Expires"
	HeaderFrom                 = "From"
	HeaderHost                 = "Host"
	HeaderIfMatch              = "If-Match"
	HeaderIfModifiedSince      = "If-Modified-Since"
	HeaderIfNoneMatch          = "If-None-Match"
	HeaderIfRange              = "If-Range"
	HeaderIfUnmodifiedSince    = "If-Unmodified-Since"
	HeaderLastModified         = "Last-Modified"
	HeaderLocation             = "Location"
	HeaderMaxForwards          = "Max-Forwards"
	HeaderOrigin               = "Origin"
	HeaderPragma               = "Pragma"
	HeaderProxyAuthenticate    = "Proxy-Authenticate"
	HeaderProxyAuthorization   = "Proxy-Authorization"
	HeaderRange                = "Range"
	HeaderReferer              = "Referer"
	HeaderRetryAfter           = "Retry-After"
	HeaderSecWebSocketKey1     = "Sec-Websocket-Key1"
	HeaderSecWebSocketKey2     = "Sec-Websocket-Key2"
	HeaderSecWebSocketProtocol = "Sec-Websocket-Protocol"
	HeaderServer               = "Server"
	HeaderSetCookie            = "Set-Cookie"
	HeaderTE                   = "Te"
	HeaderTrailer              = "Trailer"
	HeaderTransferEncoding     = "Transfer-Encoding"
	HeaderUpgrade              = "Upgrade"
	HeaderUserAgent            = "User-Agent"
	HeaderVary                 = "Vary"
	HeaderVia                  = "Via"
	HeaderWWWAuthenticate      = "Www-Authenticate"
	HeaderWarning              = "Warning"
)

// HeaderName returns the canonical format of the header name. 
func HeaderName(name string) string {
	return HeaderNameBytes([]byte(name))
}

// HeaderNameBytes returns the canonical format for the header name specified
// by the bytes in p. This function modifies the contents p.
func HeaderNameBytes(p []byte) string {
	upper := true
	for i, c := range p {
		if upper {
			if 'a' <= c && c <= 'z' {
				p[i] = c + 'A' - 'a'
			}
		} else {
			if 'A' <= c && c <= 'Z' {
				p[i] = c + 'a' - 'A'
			}
		}
		upper = c == '-'
	}
	return string(p)
}
