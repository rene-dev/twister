package main

import (
	"github.com/garyburd/twister/web"
	"template"
)

func mpGetHandler(req *web.Request) {
	mpTempl.Execute(map[string]interface{}{
		"xsrf": req.Param.GetDef(web.XSRFParamName, ""),
	},
		req.Respond(web.StatusOK, web.HeaderContentType, "text/html"))
}

func mpPostHandler(req *web.Request) {
	parts, err := web.ParseMultipartForm(req, -1)
	var (
		filename, contentType string
		contentParam          map[string]string
		size                  int
	)
	if len(parts) > 0 {
		filename = parts[0].Filename
		contentType = parts[0].ContentType
		contentParam = parts[0].ContentParam
		size = len(parts[0].Data)
	}
	mpTempl.Execute(map[string]interface{}{
		"xsrf": req.Param.GetDef(web.XSRFParamName, ""),
		"result": map[string]interface{}{
			"err":          err,
			"hello":        req.Param.GetDef("hello", ""),
			"foo":          req.Param.GetDef("foo", ""),
			"filename":     filename,
			"contentType":  contentType,
			"contentParam": contentParam,
			"size":         size,
		},
	},
		req.Respond(web.StatusOK, web.HeaderContentType, "text/html"))
}

var mpTempl = template.MustParse(mpStr, nil)

const mpStr = `
<html>
<head>
<title>muiltpart/form-data</title>
</head>
<body>
<h3>multipart/form-data</h3>
<hr>
<form method="post" action="/mp?xsrf={xsrf}" enctype="multipart/form-data">
hello <input type="text" name="hello" value="world"><br>
foo <input type="text" name="foo" value="bar"></br>
file <input type="file" name="file"></br>
<input type="submit">
</form>
{.section  result}
<hr>
err: {err}<br>
hello: {hello}<br>
foo: {foo}<br>
file name: {filename}<br>
file contentType: {contentType}<br>
file contentParam: {contentParam}<br>
file size: {size}<br>
{.end}
</body>
</html>
`
