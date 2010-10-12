package web

import (
	"bytes"
	"io"
	"http"
	"strings"
	"sort"
)

type OAuthClient struct {
	Secret          string
	Key             string
	RequestTokenURL string
	AccessTokenURL  string
	AuthorizeURL    string
}

type OAuthToken struct {
	OAuthTokenSecret string
	OAuthToken       string
}

var oauthNoEscape = [256]bool{
	'A': true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true,
	'a': true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true,
	'0': true, true, true, true, true, true, true, true, true, true,
	'-': true,
	'.': true,
	'_': true,
	'~': true,
}

// oauthEncode encodes string per RFC 5849, section 3.6. 
func oauthEncode(s string, double bool) []byte {
	// Compute size of result.
	m := 3
	if double {
		m = 5
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if oauthNoEscape[s[i]] {
			n += 1
		} else {
			n += m
		}
	}

	p := make([]byte, n)

	// Encode it.
	j := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if oauthNoEscape[b] {
			p[j] = b
			j += 1
		} else if double {
			p[j] = '%'
			p[j+1] = '2'
			p[j+2] = '5'
			p[j+3] = "0123456789ABCDEF"[b>>4]
			p[j+4] = "0123456789ABCDEF"[b&15]
			j += 5
		} else {
			p[j] = '%'
			p[j+1] = "0123456789ABCDEF"[b>>4]
			p[j+2] = "0123456789ABCDEF"[b&15]
			j += 3
		}
	}
	return p
}

type keyValueArray []struct {
	key, value []byte
}

func (p keyValueArray) Len() int {
	return len(p)
}

func (p keyValueArray) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p keyValueArray) Less(i, j int) bool {
	sgn := bytes.Compare(p[i].key, p[j].key)
	if sgn == 0 {
		sgn = bytes.Compare(p[i].value, p[j].value)
	}
	return sgn < 0
}

// writeBaseString writes method, url, and param to w using the OAuth signature
// base string compuation described in RFC 5849, section 3.4.1.
func writeBaseString(w io.Writer, method string, url string, param StringsMap) {
	// Method
	w.Write(oauthEncode(strings.ToUpper(method), false))
	w.Write([]byte{'&'})

	// URL
	parsedURL, _ := http.ParseURL(url)
	w.Write(oauthEncode(strings.ToLower(parsedURL.Scheme), false))
	w.Write(oauthEncode("://", false))
	w.Write(oauthEncode(strings.ToLower(parsedURL.Host), false))
	w.Write(oauthEncode(parsedURL.Path, false))
	w.Write([]byte{'&'})

	// Create array of parameters for sorting. For efficiency, parameters are
	// double encoded in a single step. We can do this because double encoding
	// does not change the the parameter sort order.
	n := 0
	for _, values := range param {
		n += len(values)
	}

	p := make(keyValueArray, n)
	i := 0
	for key, values := range param {
		encodedKey := oauthEncode(key, true)
		for _, value := range values {
			p[i].key = encodedKey
			p[i].value = oauthEncode(value, true)
			i += 1
		}
	}

	sort.Sort(p)

	// Write the parameters.
	amp := oauthEncode("&", false)
	equal := oauthEncode("=", false)
	sep := false
	for _, kv := range p {
		if sep {
			w.Write(amp)
		} else {
			sep = true
		}
		w.Write(kv.key)
		w.Write(equal)
		w.Write(kv.value)
	}
}
