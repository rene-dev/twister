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
	"os"
	"io"
	"strings"
	"crypto/hmac"
	"strconv"
	"time"
	"fmt"
	"encoding/hex"
)

// TimeLayout is the time layout used for HTTP headers and other values.
const TimeLayout = "Mon, 02 Jan 2006 15:04:05 GMT"

// FormatDeltaSeconds returns current time plus delta formatted per HTTP conventions.
func FormatDeltaSeconds(delta int) string {
	return time.SecondsToUTC(time.Seconds() + int64(delta)).Format(TimeLayout)
}

// FormatDeltaDays returns current time plus delta formatted per HTTP conventions.
func FormatDeltaDays(delta int) string {
	return FormatDeltaSeconds(delta * 60 * 60 * 24)
}

// Octet tyeps from RFC 2616
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

// IsTokenByte returns true if c is a token characeter as defined by RFC 2616
func IsTokenByte(c byte) bool {
	return isToken[c]
}

// IsSpaceByte returns true if c is a space characeter as defined by RFC 2616
func IsSpaceByte(c byte) bool {
	return isSpace[c]
}

// HTTP status codes from RFC 2606
const (
	StatusContinue                     = 100
	StatusSwitchingProtocols           = 101
	StatusOK                           = 200
	StatusCreated                      = 201
	StatusAccepted                     = 202
	StatusNonAuthoritativeInformation  = 203
	StatusNoContent                    = 204
	StatusResetContent                 = 205
	StatusPartialContent               = 206
	StatusMultipleChoices              = 300
	StatusMovedPermanently             = 301
	StatusFound                        = 302
	StatusSeeOther                     = 303
	StatusNotModified                  = 304
	StatusUseProxy                     = 305
	StatusTemporaryRedirect            = 307
	StatusBadRequest                   = 400
	StatusUnauthorized                 = 401
	StatusPaymentRequired              = 402
	StatusForbidden                    = 403
	StatusNotFound                     = 404
	StatusMethodNotAllowed             = 405
	StatusNotAcceptable                = 406
	StatusProxyAuthenticationRequired  = 407
	StatusRequestTimeout               = 408
	StatusConflict                     = 409
	StatusGone                         = 410
	StatusLengthRequired               = 411
	StatusPreconditionFailed           = 412
	StatusRequestEntityTooLarge        = 413
	StatusRequestURITooLong            = 414
	StatusUnsupportedMediaType         = 415
	StatusRequestedRangeNotSatisfiable = 416
	StatusExpectationFailed            = 417
	StatusInternalServerError          = 500
	StatusNotImplemented               = 501
	StatusBadGateway                   = 502
	StatusServiceUnavailable           = 503
	StatusGatewayTimeout               = 504
	StatusHTTPVersionNotSupported      = 505
)

var statusText = map[int]string{
	StatusContinue:                     "Continue",
	StatusSwitchingProtocols:           "Switching Protocols",
	StatusOK:                           "OK",
	StatusCreated:                      "Created",
	StatusAccepted:                     "Accepted",
	StatusNonAuthoritativeInformation:  "Non-Authoritative Information",
	StatusNoContent:                    "No Content",
	StatusResetContent:                 "Reset Content",
	StatusPartialContent:               "Partial Content",
	StatusMultipleChoices:              "Multiple Choices",
	StatusMovedPermanently:             "Moved Permanently",
	StatusFound:                        "Found",
	StatusSeeOther:                     "See Other",
	StatusNotModified:                  "Not Modified",
	StatusUseProxy:                     "Use Proxy",
	StatusTemporaryRedirect:            "Temporary Redirect",
	StatusBadRequest:                   "Bad Request",
	StatusUnauthorized:                 "Unauthorized",
	StatusPaymentRequired:              "Payment Required",
	StatusForbidden:                    "Forbidden",
	StatusNotFound:                     "Not Found",
	StatusMethodNotAllowed:             "Method Not Allowed",
	StatusNotAcceptable:                "Not Acceptable",
	StatusProxyAuthenticationRequired:  "Proxy Authentication Required",
	StatusRequestTimeout:               "Request Timeout",
	StatusConflict:                     "Conflict",
	StatusGone:                         "Gone",
	StatusLengthRequired:               "Length Required",
	StatusPreconditionFailed:           "Precondition Failed",
	StatusRequestEntityTooLarge:        "Request Entity Too Large",
	StatusRequestURITooLong:            "Request URI Too Long",
	StatusUnsupportedMediaType:         "Unsupported Media Type",
	StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	StatusExpectationFailed:            "Expectation Failed",
	StatusInternalServerError:          "Internal Server Error",
	StatusNotImplemented:               "Not Implemented",
	StatusBadGateway:                   "Bad Gateway",
	StatusServiceUnavailable:           "Service Unavailable",
	StatusGatewayTimeout:               "Gateway Timeout",
	StatusHTTPVersionNotSupported:      "HTTP Version Not Supported",
}

func StatusText(status int) string {
	s, found := statusText[status]
	if !found {
		s = fmt.Sprintf("Status %d", status)
	}
	return s
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

// HeaderName returns the canonical format of the header name s. 
func HeaderName(name string) string {
	p := []byte(name)
	return HeaderNameBytes(p)
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

// ProtocolVersion combines HTTP major and minor protocol numbers into a single
// integer for easy comparision.
func ProtocolVersion(major int, minor int) int {
	if minor > 999 {
		minor = 999
	}
	return major*1000 + minor
}

const (
	ProtocolVersion10 = 1000 // HTTP/1.0
	ProtocolVersion11 = 1001 // HTTP/1.1
	ContentTypeHTML   = "text/html; charset=\"utf-8\""
)

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
func ParseUrlEncodedFormBytes(p []byte, m StringsMap) os.Error {
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

func parseCookieValues(values []string, m StringsMap) os.Error {
	for _, s := range values {
		key := ""
		begin := 0
		end := 0
		for i := 0; i < len(s); i++ {
			switch s[i] {
			case ' ', '\t':
				// leading whitespace?
				if begin == end {
					begin = i + 1
					end = begin
				}
			case '=':
				key = s[begin:end]
				begin = i + 1
				end = begin
			case ';':
				if len(key) > 0 && key[0] != '$' && begin < end {
					value := s[begin:end]
					m.Append(key, value)
				}
				key = ""
				begin = i + 1
				end = begin
			default:
				end = i + 1
			}
		}
		if len(key) > 0 && key[0] != '$' && begin < end {
			m.Append(key, s[begin:end])
		}
	}
	return nil
}

func signature(secret, key, expiration, value string) string {
	hm := hmac.NewSHA1([]byte(key))
	io.WriteString(hm, key)
	hm.Write([]byte{0})
	io.WriteString(hm, expiration)
	hm.Write([]byte{0})
	io.WriteString(hm, value)
	return hex.EncodeToString(hm.Sum())
}

// SignValue timestamps and signs a string so that it cannot be forged.  
func SignValue(secret, context string, maxAgeSeconds int, value string) string {
	expiration := strconv.Itob64(time.Seconds()+int64(maxAgeSeconds), 16)
	sig := signature(secret, context, expiration, value)
	return sig + "~" + expiration + "~" + value
}

var errVerificationFailure = os.NewError("verification failed")

// VerifyValue verifies a timestamped and signed value. 
func VerifyValue(secret, context string, signedValue string) (string, os.Error) {
	a := strings.Split(signedValue, "~", 3)
	if len(a) != 3 {
		return "", errVerificationFailure
	}
	expiration, err := strconv.Btoi64(a[1], 16)
	if err != nil || expiration < time.Seconds() {
		return "", errVerificationFailure
	}
	expectedSig := signature(secret, context, a[1], a[2])
	actualSig := a[0]
	if len(actualSig) != len(expectedSig) {
		return "", errVerificationFailure
	}
	// Time independent compare
	eq := 0
	for i := 0; i < len(actualSig); i++ {
		eq = eq | (int(actualSig[i]) ^ int(expectedSig[i]))
	}
	if eq != 0 {
		return "", errVerificationFailure
	}
	return a[2], nil
}

// Cookie is an HTTP cookie. 
type Cookie struct {
	name     string
	value    string
	path     string
	domain   string
	maxAge   int
	secure   bool
	httpOnly bool
}

// NewCookie returns a new cookie with parameters name=value; path=/; httponly.
func NewCookie(name, value string) *Cookie {
	return &Cookie{name: name, value: value, path: "/", httpOnly: true}
}

// Path sets the path for the cookie. The path defaults to "/"
func (c *Cookie) Path(path string) *Cookie { c.path = path; return c }

// Domain sets the domain for the cookie. 
func (c *Cookie) Domain(domain string) *Cookie { c.domain = domain; return c }

// MaxAge sets the maximum age for the cookie in seconds.
func (c *Cookie) MaxAge(seconds int) *Cookie { c.maxAge = seconds; return c }

// MaxAgeDays sets the maximum age for the cookie in days.
func (c *Cookie) MaxAgeDays(days int) *Cookie { return c.MaxAge(days * 60 * 60 * 24) }

// Delete sets the expiration date for the cookie in the past. This will cause
// clients to delete the cookie.
func (c *Cookie) Delete() *Cookie { return c.MaxAgeDays(-30).HTTPOnly(false) }

// Secure sets the secure attribute on the cookie. The secure attribute
// defaults to false.
func (c *Cookie) Secure(secure bool) *Cookie { c.secure = secure; return c }

// Secure sets the httponly attribute on the cookie. The httponly attribute
// defaults to true.
func (c *Cookie) HTTPOnly(httpOnly bool) *Cookie { c.httpOnly = httpOnly;
return c }

// String returns the Set-Cookie header value for the cookie.
func (c *Cookie) String() string {
	var buf bytes.Buffer

	buf.WriteString(c.name)
	buf.WriteByte('=')
	buf.WriteString(c.value)

	if c.path != "" {
		buf.WriteString("; path=")
		buf.WriteString(c.path)
	}

	if c.domain != "" {
		buf.WriteString("; domain=")
		buf.WriteString(c.domain)
	}

	if c.maxAge != 0 {
		buf.WriteString("; expires=")
		buf.WriteString(FormatDeltaSeconds(c.maxAge))
	}

	if c.secure {
		buf.WriteString("; secure")
	}

	if c.httpOnly {
		buf.WriteString("; httponly")
	}

	return buf.String()
}
