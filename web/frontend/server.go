package main

import (
	"fmt"
	"net/http"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/web/proto"
)

func (s *server) serverPage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	client := discordAPIFromSession(r.Context())
	if client == nil {
		s.Sugar.Infof("Couldn't get a token from the request")
		loginRedirect(w, r)
		return
	}

	guild, err := discord.ParseSnowflake(params.ByName("id"))
	if err != nil {
		fmt.Fprint(w, "Not a server.")
		return
	}

	resp, err := s.RPC.Guild(r.Context(), &proto.GuildRequest{Id: uint64(guild), UserId: uint64(client.User.ID)})
	if err != nil {
		fmt.Fprint(w, "You're not in that server.")
		return
	}

	fmt.Fprint(w, resp.GetName())
}
