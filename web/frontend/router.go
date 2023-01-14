package frontend

import (
	"embed"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/russross/blackfriday/v2"
	"github.com/starshine-sys/catalogger/common"
)

//go:embed static/*
var static embed.FS

//go:embed static/privacy.md
var privacyMD []byte

//go:embed static/docs.md
var docsMD []byte

//go:embed static/tos.md
var tosMD []byte

//go:embed static/contact.md
var contactMD []byte

func newRouter(s *server) chi.Router {
	r := chi.NewMux()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", s.index)
	r.Get("/login", s.handleLogin)
	r.Get("/authorize", s.handleAuthorize)

	r.Route("/servers", func(r chi.Router) {
		r.Use(s.RequireSession)

		r.Get("/", s.serverList)
		r.Get(`/{id:\d+}`, s.serverPage)

		r.Post(`/{id:\d+}/save-channels`, s.saveChannels)
		r.Post(`/{id:\d+}/save-ignored`, s.saveIgnored)

		r.Post(`/{id:\d+}/add-redirect`, s.addRedirect)
		r.Post(`/{id:\d+}/delete-redirect/{channel:\d+}`, s.delRedirect)
	})

	r.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		c := &http.Cookie{
			Name:    sessionCookieName,
			Value:   "",
			Expires: time.Now(),
			Path:    "/",
		}

		http.SetCookie(w, c)

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	r.Get("/privacy", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Privacy template.HTML
		}{}

		data.Privacy = template.HTML(blackfriday.Run(privacyMD))

		err := tmpl.ExecuteTemplate(w, "privacy.html", data)
		if err != nil {
			common.Log.Errorf("Error executing template: %v", err)
			return
		}
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Docs template.HTML
		}{}

		data.Docs = template.HTML(blackfriday.Run(docsMD))

		err := tmpl.ExecuteTemplate(w, "docs.html", data)
		if err != nil {
			common.Log.Errorf("Error executing template: %v", err)
			return
		}
	})

	r.Get("/tos", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Docs template.HTML
		}{}

		data.Docs = template.HTML(blackfriday.Run(tosMD))

		err := tmpl.ExecuteTemplate(w, "tos.html", data)
		if err != nil {
			common.Log.Errorf("Error executing template: %v", err)
			return
		}
	})

	r.Get("/contact", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Docs template.HTML
		}{}

		data.Docs = template.HTML(blackfriday.Run(contactMD))

		err := tmpl.ExecuteTemplate(w, "tos.html", data)
		if err != nil {
			common.Log.Errorf("Error executing template: %v", err)
			return
		}
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, `User-agent: *
Disallow: /servers/
Disallow: /login
Disallow: /authorize`)
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		s.error(w, http.StatusNotFound, false, "You've taken a wrong turn! No page exists at this address. Try again from the home page, maybe?")
	})

	r.Mount("/static", http.FileServer(http.FS(static)))

	return r
}
