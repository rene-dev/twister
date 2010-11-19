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
	"bytes"
	"mime"
)

var (
	errMpFraming  = os.NewError("twister: bad framing in multipart/form-data")
	errMpHeader   = os.NewError("twsiter: bad multipart/form-data header")
	dashdashBytes = []byte("--")
	crlfBytes     = []byte("\r\n")
)

type Part struct {
	Name         string
	Filename     string
	ContentType  string
	ContentParam map[string]string
	Data         []byte
}

// ParseMultipartForm parses a multipart/form-data body. Form fields are
// added to the request Param. This function loads the entire request body in
// memory. This may not be appropriate in some scenarios.
func ParseMultipartForm(req *Request, maxRequestBodyLen int) ([]Part, os.Error) {
	if req.ContentType != "multipart/form-data" {
		return nil, nil
	}
	p, err := req.BodyBytes(maxRequestBodyLen)
	if err != nil {
		return nil, err
	}
	boundary := req.ContentParam["boundary"]
	if boundary == "" {
		return nil, os.NewError("twister: multipart/form-data boundary missing")
	}

	sep := []byte("\r\n--" + boundary)

	// Position after first separator.
	if bytes.HasPrefix(p, sep[2:]) {
		p = p[len(sep)-2:]
	} else {
		i := bytes.Index(p, sep)
		if i < 0 {
			return nil, errMpFraming
		}
		p = p[i:]
	}

	var result []Part

	for {
		// Handle trailing "--" or "\r\n"
		if len(p) < 2 {
			return nil, errMpFraming
		}
		if bytes.HasPrefix(p, dashdashBytes) {
			break
		}
		if !bytes.HasPrefix(p, crlfBytes) {
			return nil, errMpFraming
		}

		// Split off part
		i := bytes.Index(p, sep)
		if i < 0 {
			return nil, errMpFraming
		}
		part := p[2:i]
		p = p[i+len(sep):]

		var contentType string
		var contentParam map[string]string
		var filename string
		var name string

		// Loop over header lines
		for {
			i := bytes.Index(part, crlfBytes)
			if i < 0 {
				return nil, errMpHeader
			}
			line := part[:i]
			part = part[i+2:]
			if len(line) == 0 {
				break
			}

			// Parse line to lowercase key and value.
			var key, value string
			for i, b := range line {
				if b == ':' {
					key = string(line[:i])
					value = string(line[i+1:])
					break
				} else if mime.IsTokenChar(int(b)) {
					if 'A' <= b && b <= 'Z' {
						line[i] = b + 'a' - 'A'
					}
				} else {
					return nil, errMpHeader
				}
			}

			switch key {
			case "content-type":
				contentType, contentParam = mime.ParseMediaType(value)
			case "content-disposition":
				disposition, param := mime.ParseMediaType(value)
				if disposition != "form-data" {
					continue
				}
				name = param["name"]
				filename = param["filename"]
			}
		}

		if name == "" {
			continue
		}

		if filename == "" {
			req.Param.Append(name, string(part))
		} else {
			result = append(result, Part{
				ContentType:  contentType,
				ContentParam: contentParam,
				Name:         name,
				Filename:     filename,
				Data:         part})
		}
	}
	return result, nil
}
