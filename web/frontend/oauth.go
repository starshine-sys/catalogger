package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/julienschmidt/httprouter"
	"github.com/mediocregopher/radix/v4"
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

// literally only used to add the bot and applications.commands scopes
var oauthBotConfig = oauth2.Config{
	Scopes: []string{"identify", "guilds", "bot", "applications.commands"},
}

func init() {
	https, _ := strconv.ParseBool(os.Getenv("HTTPS"))
	if https {
		oauthConfig.RedirectURL = "https://" + os.Getenv("HOST") + "/authorize"
	} else {
		oauthConfig.RedirectURL = "http://" + os.Getenv("HOST") + "/authorize"
	}

	oauthBotConfig.ClientID = oauthConfig.ClientID
	oauthBotConfig.ClientSecret = oauthConfig.ClientSecret
	oauthBotConfig.Endpoint = oauthConfig.Endpoint
}

func (s *server) createCSRFToken(ctx context.Context) (token string, err error) {
	token = RandBase64(32)

	err = s.multiRedis(ctx,
		radix.Cmd(nil, "LPUSH", "csrf", token),
		radix.Cmd(nil, "LTRIM", "csrf", "0", "999"),
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *server) checkCSRFToken(ctx context.Context, token string) (matched bool, err error) {
	var num int
	err = s.Redis.Do(ctx, radix.Cmd(&num, "LREM", "csrf", "1", token))
	if err != nil {
		return
	}
	return num > 0, nil
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	csrf, err := s.createCSRFToken(ctx)
	if err != nil {
		s.Sugar.Errorf("Error setting CSRF token: %v", err)
		return
	}

	redir := r.FormValue("goto")
	if redir == "" {
		redir = "/servers"
	}

	err = s.Redis.Do(ctx, radix.Cmd(nil, "SET", "csrf-redir:"+csrf, redir, "EX", "600"))
	if err != nil {
		s.Sugar.Errorf("Error setting redirect: %v", err)
	}

	url := oauthConfig.AuthCodeURL(csrf, oauth2.AccessTypeOnline) + "&prompt=none"

	if r.FormValue("add") != "" {
		url = oauthBotConfig.AuthCodeURL(csrf, oauth2.AccessTypeOnline) + "&prompt=none&permissions=537259248"
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *server) handleAuthorize(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	state := r.FormValue("state")

	if ok, err := s.checkCSRFToken(ctx, state); !ok {
		if err != nil {
			s.Sugar.Errorf("Error validating state: %v", err)
		} else {
			s.Sugar.Infof("Invalid state %v", state)
		}
		http.Redirect(w, r, "/?error=bad-csrf", http.StatusTemporaryRedirect)
		return
	}

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

	var redir string
	err = s.Redis.Do(ctx, radix.Cmd(&redir, "GET", "csrf-redir:"+state))
	if err != nil {
		redir = "/servers"
	} else {
		s.Redis.Do(ctx, radix.Cmd(nil, "DEL", "csrf-redir:"+state))
	}

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

		cache, ok := s.getUser(cookie.Value)
		if !ok {
			client := api.NewClient(t.Type() + " " + t.AccessToken)
			u, err := client.Me()
			if err != nil {
				s.Sugar.Errorf("Error getting Discord user: %v", err)
				loginRedirect(w, r)
				return
			}

			cache = &userCache{
				Client: client,
				User:   u,
			}
			s.setUser(cookie.Value, cache)
		}

		ctx = context.WithValue(ctx, contextKeyDiscord, cache)

		inner(w, r.Clone(ctx), params)
	}
	return httprouter.Handle(mw)
}

func loginRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login?goto="+url.QueryEscape(r.RequestURI), http.StatusTemporaryRedirect)
}
