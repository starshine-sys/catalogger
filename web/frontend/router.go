package main

import (
	"embed"
	"html/template"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/russross/blackfriday/v2"
)

//go:embed static/*
var static embed.FS

//go:embed static/privacy.md
var privacyMD []byte

//go:embed static/docs.md
var docsMD []byte

func newRouter(s *server) *httprouter.Router {
	r := httprouter.New()

	r.GET("/", s.index)
	r.GET("/login", s.handleLogin)
	r.GET("/authorize", s.handleAuthorize)
	r.GET("/servers", s.RequireSession(s.serverList))
	r.GET("/servers/:id", s.RequireSession(s.serverPage))

	r.POST("/servers/:id/save-channels", s.RequireSession(s.saveChannels))
	r.POST("/servers/:id/save-ignored", s.RequireSession(s.saveIgnored))
	r.POST("/servers/:id/add-redirect", s.RequireSession(s.addRedirect))
	r.POST("/servers/:id/delete-redirect/:channel", s.RequireSession(s.delRedirect))

	r.GET("/logout", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		c := &http.Cookie{
			Name:    sessionCookieName,
			Value:   "",
			Expires: time.Now(),
			Path:    "/",
		}

		http.SetCookie(w, c)

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	r.GET("/privacy", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		data := struct {
			Privacy template.HTML
		}{}

		data.Privacy = template.HTML(blackfriday.Run(privacyMD))

		err := tmpl.ExecuteTemplate(w, "privacy.html", data)
		if err != nil {
			s.Sugar.Errorf("Error executing template: %v", err)
			return
		}
	})

	r.GET("/docs", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		data := struct {
			Docs template.HTML
		}{}

		data.Docs = template.HTML(blackfriday.Run(docsMD))

		err := tmpl.ExecuteTemplate(w, "docs.html", data)
		if err != nil {
			s.Sugar.Errorf("Error executing template: %v", err)
			return
		}
	})

	r.ServeFiles("/static/*filepath", http.FS(static))

	return r
}
