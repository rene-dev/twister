package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/twister/oauth"
	"github.com/garyburd/twister/server"
	"github.com/garyburd/twister/web"
	"http"
	"log"
	"os"
	"strings"
	"template"
)

var oauthClient = oauth.Client{
	Credentials: oauth.Credentials{
		"bD3jC40mGxqQtVHomJolg",
		"WWCbD3zYL8Y98WKBauejx93VYqS0kBaHjOcp3PQtw"},
	TemporaryCredentialRequestURI: "http://api.twitter.com/oauth/request_token",
	ResourceOwnerAuthorizationURI: "http://api.twitter.com/oauth/authenticate",
	TokenRequestURI:               "http://api.twitter.com/oauth/access_token",
}

// encodeCredentials encodes OAuth credentials in a format suitable for storing in a cookie.
func encodeCredentials(c *oauth.Credentials) string {
	return http.URLEscape(c.Token) + "/" + http.URLEscape(c.Secret)
}

// credentials returns oauth credentials stored in cookie with name key.
func credentials(req *web.Request, key string) (*oauth.Credentials, os.Error) {
	s, found := req.Cookie.Get(key)
	if !found {
		return nil, os.NewError("main: missing cookie")
	}
	a := strings.Split(s, "/", -1)
	if len(a) != 2 {
		return nil, os.NewError("main: bad credential cookie")
	}
	token, err := http.URLUnescape(a[0])
	if err != nil {
		return nil, os.NewError("main: bad credential cookie")
	}
	secret, err := http.URLUnescape(a[1])
	if err != nil {
		return nil, os.NewError("main: bad credential cookie")
	}
	return &oauth.Credentials{token, secret}, nil
}

func login(req *web.Request) {
	temporaryCredentials, err := oauthClient.RequestTemporaryCredentials("")
	if err != nil {
		req.Error(web.StatusInternalServerError, err)
		return
	}
	req.Redirect(oauthClient.AuthorizationURL(temporaryCredentials), false,
		web.HeaderSetCookie, fmt.Sprintf("tc=%s; Path=/; HttpOnly", encodeCredentials(temporaryCredentials)))
}

func twitterCallback(req *web.Request) {
	temporaryCredentials, err := credentials(req, "tc")
	if err != nil {
		req.Error(web.StatusNotFound, err)
		return
	}
	s, found := req.Param.Get("oauth_token")
	if !found {
		req.Error(web.StatusNotFound, os.NewError("main: no token"))
		return
	}
	if s != temporaryCredentials.Token {
		req.Error(web.StatusNotFound, os.NewError("main: token mismatch"))
		return
	}
	tokenCredentials, _, err := oauthClient.RequestToken(temporaryCredentials)
	if s != temporaryCredentials.Token {
		req.Error(web.StatusNotFound, err)
		return
	}
	req.Redirect("/", false,
		web.HeaderSetCookie, fmt.Sprintf("auth=%s; Path=/; HttpOnly", encodeCredentials(tokenCredentials)))
}


func home(req *web.Request) {

}

func main() {
	flag.Parse()
	h := web.ProcessForm(10000, true, web.NewRouter().
		Register("/", "GET", home).
		Register("/login", "GET", login).
		Register("/account/twitter-callback", "GET", twitterCallback))

	err := server.ListenAndServe("localhost:8080", ":8080", h)
	if err != nil {
		log.Exit("ListenAndServe:", err)
	}
}

var homeTempl = template.MustParse(homeStr, nil)

const homeStr = `
<html>
<head>
</head>
<body>
<ul>
<li><a href="/core">Core functionality</a>
<li><a href="/chat">Chat using WebSockets</a>
</ul>
</body>
</html>`
