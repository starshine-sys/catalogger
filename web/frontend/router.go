package main

import "github.com/julienschmidt/httprouter"

func newRouter(s *server) *httprouter.Router {
	r := httprouter.New()

	r.GET("/", nil)
	r.GET("/login", s.handleLogin)
	r.GET("/authorize", s.handleAuthorize)
	r.GET("/servers", s.RequireSession(s.serverList))
	r.GET("/servers/:id", s.RequireSession(s.serverPage))

	r.POST("/servers/:id/save-channels", nil)
	r.POST("/servers/:id/save-ignored", nil)
	r.POST("/servers/:id/add-redirect", nil)
	r.POST("/servers/:id/delete-redirect/:channel", nil)

	return r
}
