// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package guestbook

import (
	"io"
	"os"
	"template"
	"time"
	"github.com/garyburd/twister/gae"
	"github.com/garyburd/twister/web"
	"appengine/datastore"
	"appengine/user"
)

type Greeting struct {
	Author  string
	Content string
	Date    datastore.Time
}

func serveError(req *web.Request, status int, reason os.Error, header web.Header) {
	header.Set(web.HeaderContentType, "text/plain; charset=utf-8")
	w := req.Responder.Respond(status, header)
	io.WriteString(w, web.StatusText(status))
	if reason != nil {
		io.WriteString(w, "\n")
		io.WriteString(w, reason.String())
	}
}

var mainPage = template.MustParse(`<html><body>
{.repeated section gg}
{.section Author}<b>{@|html}</b>{.or}An anonymous person{.end}
wrote <blockquote>{Content|html}</blockquote>
{.end}
<form action="/sign" method="POST">
<div><input type="hidden" name="xsrf" value="{xsrf}"> <textarea name="content" rows="3" cols="60"></textarea></div>
<div><input type="submit" value="Sign Guestbook"></div>
</form></body></html>`,
	nil)

func handleMainPage(r *web.Request) {
	c := gae.Context(r)
	q := datastore.NewQuery("Greeting").Order("-Date").Limit(10)
	var gg []*Greeting
	_, err := q.GetAll(c, &gg)
	if err != nil {
		r.Error(web.StatusInternalServerError, err)
		return
	}
	w := r.Respond(200, "Content-Type", "text/html")
	if err := mainPage.Execute(w, map[string]interface{}{
		"xsrf": r.Param.Get("xsrf"),
		"gg":   gg}); err != nil {
		c.Logf("%v", err)
	}
}

func handleSign(r *web.Request) {
	c := gae.Context(r)
	g := &Greeting{
		Content: r.Param.Get("content"),
		Date:    datastore.SecondsToTime(time.Seconds()),
	}
	if u := user.Current(c); u != nil {
		g.Author = u.String()
	}
	if _, err := datastore.Put(c, datastore.NewIncompleteKey("Greeting"), g); err != nil {
		r.Error(web.StatusInternalServerError, err)
		return
	}
	r.Redirect("/", false)
}

func init() {
	gae.Handle("/",
		web.SetErrorHandler(serveError,
			web.FormHandler(1000, true,
				web.NewRouter().
					Register("/", "GET", handleMainPage).
					Register("/sign", "POST", handleSign))))
}
