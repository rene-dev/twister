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

// This application displays a user's Facebook news feed.  
package main

// This code does not handle errors from Facebook gracefully.

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/garyburd/twister/server"
	"github.com/garyburd/twister/web"
	"http"
	"io/ioutil"
	"json"
	"log"
	"os"
	"strconv"
)

var appID string
var appSecret string

// getUrlEncodedForm fetches a URL and decodes the response body as a URL encoded form.
func getUrlEncodedForm(url string, param web.Values) (web.Values, os.Error) {
	if param != nil {
		url = url + "?" + param.FormEncodedString()
	}
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, os.NewError(fmt.Sprint("Status ", r.StatusCode))
	}
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	m := make(web.Values)
	err = m.ParseFormEncodedBytes(p)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// getJSON fetches a URL and decodes the response body as JSON.
func getJSON(url string, param web.Values) (interface{}, os.Error) {
	if param != nil {
		url = url + "?" + param.FormEncodedString()
	}
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, os.NewError(fmt.Sprint("Status ", r.StatusCode))
	}
	p, _ := ioutil.ReadAll(r.Body)
	var v interface{}
	err = json.NewDecoder(bytes.NewBuffer(p)).Decode(&v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// acccessToken returns OAuth2 access token stored in a cookie.
func accessToken(req *web.Request) (string, os.Error) {
	s := req.Cookie.Get("fbtok")
	if s == "" {
		return "", os.NewError("main: missing cookie")
	}
	token, err := http.URLUnescape(s)
	if err != nil {
		return "", os.NewError("main: bad credential cookie")
	}
	return token, nil
}

// loginHandler redirects to Facebook OAuth2 authorization page.
func loginHandler(req *web.Request) {
	m := web.NewValues(
		"client_id", appID, // defined in settings.go
		"scope", "read_stream",
		"redirect_uri", req.URL.Scheme+"://"+req.URL.Host+"/callback")
	req.Redirect("https://graph.facebook.com/oauth/authorize?"+m.FormEncodedString(), false)
}

// logoutHandler logs the user out by clearing the access token cookie.
func logoutHandler(req *web.Request) {
	req.Redirect("/", false,
		web.HeaderSetCookie, web.NewCookie("fbtok", "").Delete().String())
}

// authCallbackHandler handles redirect from Facebook OAuth2 authorization page.
func authCallbackHandler(req *web.Request) {
	code := req.Param.Get("code")
	if code == "" {
		// should display error_reason
		req.Redirect("/", false)
		return
	}
	f, err := getUrlEncodedForm("https://graph.facebook.com/oauth/access_token",
		web.NewValues(
			"client_id", appID, // defined in settings.go
			"client_secret", appSecret, // defined in settings.go
			"redirect_uri", req.URL.Scheme+"://"+req.URL.Host+"/callback",
			"code", code))
	if err != nil {
		req.Error(web.StatusInternalServerError, err)
		return
	}
	token := f.Get("access_token")
	expires := f.Get("expires")
	if expires == "" {
		expires = "3600"
	}
	maxAge, err := strconv.Atoi(expires)
	if err != nil {
		maxAge = 3600
	} else {
		maxAge -= 30 // fudge
	}
	req.Redirect("/", false,
		web.HeaderSetCookie, web.NewCookie("fbtok", token).
			MaxAge(maxAge-30).String())
}

// loggedOutHandler handles request to the home page for logged out users.
func loggedOutHandler(req *web.Request) {
	loggedOutTemplate.respond(req, web.StatusOK, nil)
}

// home handles requests to the home page.
func homeHandler(req *web.Request) {
	token, err := accessToken(req)
	if err != nil {
		loggedOutHandler(req)
		return
	}
	feed, err := getJSON("https://graph.facebook.com/me/home", web.NewValues("access_token", token))
	if err != nil {
		req.Error(web.StatusInternalServerError, err,
			web.HeaderSetCookie, web.NewCookie("fbtok", "").Delete().String())
		return
	}
	homeTemplate.respond(req, web.StatusOK, feed)
}

func readSettings() {
	b, err := ioutil.ReadFile("settings.json")
	if err != nil {
		log.Fatal("could not read settings.json", err)
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		log.Fatal("could not unmarhal settings.json", err)
	}
	appID = m["AppID"].(string)
	appSecret = m["AppSecret"].(string)
}

func main() {
	flag.Parse()
	readSettings()
	h := web.FormHandler(10000, true, web.NewRouter().
		Register("/", "GET", homeHandler).
		Register("/logout", "GET", logoutHandler).
		Register("/login", "GET", loginHandler).
		Register("/callback", "GET", authCallbackHandler))

	server.Run(":8080", h)
}
