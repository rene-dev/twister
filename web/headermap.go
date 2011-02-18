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
	"io"
	"bufio"
	"strings"
	"bytes"
)

// Octet types from RFC 2616
var (
	isText  [256]bool
	isToken [256]bool
	isSpace [256]bool
)

func init() {
	// OCTET      = <any 8-bit sequence of data>
	// CHAR       = <any US-ASCII character (octets 0 - 127)>
	// CTL        = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
	// CR         = <US-ASCII CR, carriage return (13)>
	// LF         = <US-ASCII LF, linefeed (10)>
	// SP         = <US-ASCII SP, space (32)>
	// HT         = <US-ASCII HT, horizontal-tab (9)>
	// <">        = <US-ASCII double-quote mark (34)>
	// CRLF       = CR LF
	// LWS        = [CRLF] 1*( SP | HT )
	// TEXT       = <any OCTET except CTLs, but including LWS>
	// separators = "(" | ")" | "<" | ">" | "@" | "," | ";" | ":" | "\" | <"> 
	//              | "/" | "[" | "]" | "?" | "=" | "{" | "}" | SP | HT
	// token      = 1*<any CHAR except CTLs or separators>
	// qdtext     = <any TEXT except <">>

	for c := 0; c < 256; c++ {
		isCtl := (0 <= c && c <= 31) || c == 127
		isChar := 0 <= c && c <= 127
		isSpace[c] = strings.IndexRune(" \t\r\n", c) >= 0
		isSeparator := strings.IndexRune(" \t\"(),/:;<=>?@[]\\{}", c) >= 0
		isText[c] = isSpace[c] || !isCtl
		isToken[c] = isChar && !isCtl && !isSeparator
	}
}

// IsTokenByte returns true if c is a token character as defined by RFC 2616
func IsTokenByte(c byte) bool {
	return isToken[c]
}

// IsSpaceByte returns true if c is a space character as defined by RFC 2616
func IsSpaceByte(c byte) bool {
	return isSpace[c]
}

var (
	ErrLineTooLong    = os.NewError("HTTP header line too long")
	ErrBadHeaderLine  = os.NewError("could not parse HTTP header line")
	ErrHeaderTooLong  = os.NewError("HTTP header value too long")
	ErrHeadersTooLong = os.NewError("too many HTTP headers")
)

// HeaderMap maps canonical header names to slices of strings. Use the
// functions HeaderName and HeaderNameBytes to convert names to canonical
// format.
type HeaderMap map[string][]string

// NewHeaderMap returns a map initialized with the given key-value pairs.
func NewHeaderMap(kvs ...string) HeaderMap {
	if len(kvs)%2 == 1 {
		panic("twister: even number args required for NewHeaderMap")
	}
	m := HeaderMap{}
	for i := 0; i < len(kvs); i += 2 {
		m.Add(kvs[i], kvs[i+1])
	}
	return m
}

// Get returns the first value for given key or "" if the key is not found.
func (m HeaderMap) Get(key string) string {
	values, found := m[key]
	if !found || len(values) == 0 {
		return ""
	}
	return values[0]
}

// Add appends value to slice for given key.
func (m HeaderMap) Add(key string, value string) {
	m[key] = append(m[key], value)
}

// Set value for given key, discarding previous values if any.
func (m HeaderMap) Set(key string, value string) {
	m[key] = []string{value}
}

// GetList returns list of comma separated values over multiple header values
// for the given key. Commas are ignored in quoted strings. Quoted values are
// not unescaped or unqoted. Whitespace is trimmmed.
func (m HeaderMap) GetList(key string) []string {
	var result []string
	for _, s := range m[key] {
		begin := 0
		end := 0
		escape := false
		quote := false
		for i := 0; i < len(s); i++ {
			b := s[i]
			switch {
			case escape:
				escape = false
				end = i + 1
			case quote:
				switch b {
				case '\\':
					escape = true
				case '"':
					quote = false
				}
				end = i + 1
			case b == '"':
				quote = true
				end = i + 1
			case isSpace[b]:
				if begin == end {
					begin = i + 1
					end = begin
				}
			case b == ',':
				result = append(result, s[begin:end])
				begin = i + 1
				end = begin
			default:
				end = i + 1
			}
		}
		if begin < end {
			result = append(result, s[begin:end])
		}
	}
	return result
}

// WriteHttpHeader writes the map in HTTP header format.
func (m HeaderMap) WriteHttpHeader(w io.Writer) os.Error {
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
			// Convert \r and \n to space to prevent response splitting attacks.
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

// ParseHttpHeader parses the HTTP headers and appends the values to the
// supplied map. Header names are converted to canonical format.
func (m HeaderMap) ParseHttpHeader(b *bufio.Reader) (err os.Error) {

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
			m.Add(key, value)
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

// Header names in canonical format.
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
	HeaderXXSRFToken           = "X-Xsrftoken"
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

// QuoteHeaderValue quotes s using quoted-string rules described in RFC 2616.
func QuoteHeaderValue(s string) string {
	var b bytes.Buffer
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"':
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

// QuoteHeaderValueOrToken quotes s if s is not a valid token per RFC 2616.
func QuoteHeaderValueOrToken(s string) string {
	for i := 0; i < len(s); i++ {
		if !isToken[s[i]] {
			return QuoteHeaderValue(s)
		}
	}
	return s
}

// UnquoteHeaderValue unquotes s if s is surrounded by quotes, otherwise s is
// returned.
func UnquoteHeaderValue(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	s = s[1 : len(s)-1]
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			var buf bytes.Buffer
			buf.WriteString(s[:i])
			escape := true
			for j := i + 1; j < len(s); j++ {
				b := s[j]
				switch {
				case escape:
					escape = false
					buf.WriteByte(b)
				case b == '\\':
					escape = true
				default:
					buf.WriteByte(b)
				}
			}
			s = buf.String()
			break
		}
	}
	return s
}
