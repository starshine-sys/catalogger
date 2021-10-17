package main

import (
	"net/http"
	"os"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/web/proto"
)

const newsRefresh = time.Hour

func (s *server) index(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := r.Context()

	var guildCount, userCount int64
	resp, err := s.RPC.GuildUserCount(ctx, &proto.GuildUserCountRequest{})
	if err == nil {
		guildCount = resp.GetGuildCount()
		userCount = resp.GetUserCount()
	}

	var news []discord.Message
	if len(s.news) > 0 {
		news = append(news, s.news...)
	}

	data := struct {
		SupportServer string
		ClientID      string
		GuildCount    string
		UserCount     string
		News          []discord.Message
	}{
		SupportServer: os.Getenv("SUPPORT_SERVER"),
		ClientID:      oauthConfig.ClientID,
		GuildCount:    humanize.Comma(guildCount),
		UserCount:     humanize.Comma(userCount),
		News:          news,
	}

	if s.newsFetchTime.Before(time.Now().Add(-newsRefresh)) {
		go s.fetchNews()
	}

	err = tmpl.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		s.Sugar.Errorf("Error executing template: %v", err)
		return
	}
}

func (s *server) fetchNews() {
	if s.newsClient == nil || os.Getenv("TOKEN") == "" {
		return
	}

	news, err := s.newsClient.Messages(s.newsChannel, 5)
	if err != nil {
		s.Sugar.Errorf("Couldn't fetch news messages: %v", err)
	} else {
		s.news = news
	}
	s.newsFetchTime = time.Now()
}
