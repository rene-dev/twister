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
	errMpFraming = os.NewError("twister: bad framing in multipart/form-data")
	errMpHeader  = os.NewError("twsiter: bad multipart/form-data header")
)

// Part represents an element of a multi-part request entity.
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

		header := make(HeaderMap)
		n, err := header.ParseHttpHeaderBytes(part)
		if err != nil {
			return nil, err
		}
		part = part[n:]

		if val := header.Get(HeaderContentDisposition); val != "" {
			disposition, dispositionParam := mime.ParseMediaType(val)
			if disposition == "form-data" {
				if name := dispositionParam["name"]; name != "" {
					if filename := dispositionParam["filename"]; filename != "" {
						contentType, contentParam := mime.ParseMediaType(header.Get(HeaderContentType))
						result = append(result, Part{
							ContentType:  contentType,
							ContentParam: contentParam,
							Name:         name,
							Filename:     filename,
							Data:         part})
					} else {
						req.Param.Add(name, string(part))
					}
				}
			}
		}
	}
	return result, nil
}
