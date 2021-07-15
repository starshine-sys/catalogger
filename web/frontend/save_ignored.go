package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/web/proto"
)

func (s *server) saveIgnored(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := r.Context()

	client := discordAPIFromSession(ctx)
	if client == nil {
		s.Sugar.Infof("Couldn't get a token from the request")
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	guildID, err := discord.ParseSnowflake(params.ByName("id"))
	if err != nil {
		http.Error(w, "Not a server", http.StatusNotFound)
		return
	}

	resp, err := s.RPC.Guild(r.Context(), &proto.GuildRequest{Id: uint64(guildID), UserId: uint64(client.User.ID)})
	if err != nil || !discord.Permissions(resp.GetPermissions()).Has(discord.PermissionManageGuild) {
		http.Error(w, "Missing permissions.", http.StatusUnauthorized)
		return
	}

	err = r.ParseForm()
	if err != nil {
		s.Sugar.Errorf("Error parsing form: %v", err)
		http.Error(w, "Error parsing form", http.StatusInternalServerError)
	}

	for k, v := range r.Form {
		fmt.Println(k, ":", v)
	}

	channels := []uint64{}
	for _, entry := range r.Form["ignored-channels"] {
		id, err := strconv.ParseUint(entry, 10, 64)
		if err != nil {
			s.Sugar.Infof("Invalid channel id `%v`", entry)
			id = 0
		}

		isChannel := false
		for _, ch := range resp.GetChannels() {
			if ch.GetId() == id {
				isChannel = true
				break
			}
		}

		if isChannel {
			channels = append(channels, id)
		}
	}

	fmt.Println(r.Form["ignored-channels"])
	fmt.Println(channels)

	_, err = s.DB.Pool.Exec(ctx, "update guilds set ignored_channels = $1 where id = $2", channels, resp.GetId())
	if err != nil {
		s.Sugar.Errorf("Error setting ignored channels: %v", err)
		http.Error(w, "Error setting channels.", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Success!")
	return
}