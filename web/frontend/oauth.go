package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2"
)

var sessionCookieName = "catalogger-session"

var oauthConfig = oauth2.Config{
	ClientID:     os.Getenv("CLIENT_ID"),
	ClientSecret: os.Getenv("CLIENT_SECRET"),
	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://discord.com/api/oauth2/authorize",
		TokenURL:  "https://discord.com/api/oauth2/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"identify", "guilds"},
}

func init() {
	https, _ := strconv.ParseBool(os.Getenv("HTTPS"))
	if https {
		oauthConfig.RedirectURL = "https://" + os.Getenv("HOST") + "/authorize"
	} else {
		oauthConfig.RedirectURL = "http://" + os.Getenv("HOST") + "/authorize"
	}
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	csrf := RandBase64(32)
	s.CSRFTokens.Set(csrf, r.FormValue("goto"))

	url := oauthConfig.AuthCodeURL(csrf, oauth2.AccessTypeOnline) + "&prompt=none"

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *server) handleAuthorize(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	state := r.FormValue("state")
	v, err := s.CSRFTokens.Get(state)
	if err != nil {
		if err == ttlcache.ErrNotFound {
			fmt.Fprintln(w, "Couldn't validate your CSRF token.")
			return
		}

		s.Sugar.Errorf("Error validating state: %v", err)
		return
	}
	redir := v.(string)
	if redir == "" {
		redir = "/servers"
	}

	s.CSRFTokens.Remove(state)

	ctx := r.Context()
	code := r.FormValue("code")
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		s.Sugar.Errorf("Error exchanging code for token: %v", err)
		return
	}

	cookie, err := CookieFromToken(token)
	if err != nil {
		s.Sugar.Errorf("Error storing auth token: %v", err)
		return
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, redir, http.StatusTemporaryRedirect)
}

// CookieFromToken creates a cookie from a token
func CookieFromToken(token *oauth2.Token) (c *http.Cookie, err error) {
	token.RefreshToken = ""

	data, err := json.Marshal(token)
	if err != nil {
		return
	}

	expiry := 30 * 24 * time.Hour
	if time.Until(token.Expiry) < expiry {
		expiry = time.Until(token.Expiry)
	}

	c = &http.Cookie{
		Name:   sessionCookieName,
		Value:  base64.URLEncoding.EncodeToString(data),
		MaxAge: int(expiry.Seconds()),
		Path:   "/",
	}

	return c, nil
}

// RandBase64 ...
func RandBase64(size int) string {
	b := make([]byte, size)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(b)
}

var errTokenExpired = errors.New("token is expired")

func (s *server) RequireSession(inner httprouter.Handle) httprouter.Handle {
	mw := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := r.Context()

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			s.Sugar.Errorf("Error getting cookie: %v", err)
			loginRedirect(w, r)
			return
		}

		b, err := base64.URLEncoding.DecodeString(cookie.Value)
		if err != nil {
			s.Sugar.Errorf("Error decoding cookie: %v", err)
			loginRedirect(w, r)
			return
		}

		t := &oauth2.Token{}

		err = json.Unmarshal(b, t)
		if err != nil {
			s.Sugar.Errorf("Error unmarshaling cookie: %v", err)
			loginRedirect(w, r)
			return
		}

		client := api.NewClient(t.Type() + " " + t.AccessToken)

		ctx = context.WithValue(ctx, contextKeyDiscord, client)

		inner(w, r.Clone(ctx), params)
	}
	return httprouter.Handle(mw)
}

func loginRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login?goto="+url.QueryEscape(r.RequestURI), http.StatusTemporaryRedirect)
}
