package main

import (
	"net/http"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/web/proto"
)

func (s *server) index(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := r.Context()

	var guildCount, userCount int64
	resp, err := s.RPC.GuildUserCount(ctx, &proto.GuildUserCountRequest{})
	if err == nil {
		guildCount = resp.GetGuildCount()
		userCount = resp.GetUserCount()
	}

	data := struct {
		SupportServer string
		ClientID      string
		GuildCount    string
		UserCount     string
	}{
		SupportServer: os.Getenv("SUPPORT_SERVER"),
		ClientID:      oauthConfig.ClientID,
		GuildCount:    humanize.Comma(guildCount),
		UserCount:     humanize.Comma(userCount),
	}

	err = tmpl.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		s.Sugar.Errorf("Error executing template: %v", err)
		return
	}
}
