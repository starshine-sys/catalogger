package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/web/proto"
)

func (s *server) saveChannels(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := r.Context()

	client := discordAPIFromSession(ctx)
	if client == nil {
		common.Log.Infof("Couldn't get a token from the request")
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	guildID, err := discord.ParseSnowflake(params.ByName("id"))
	if err != nil {
		http.Error(w, "Not a server", http.StatusNotFound)
		return
	}

	resp, err := s.rpcGuild(ctx, discord.GuildID(guildID), client)
	if err != nil || !discord.Permissions(resp.GetPermissions()).Has(discord.PermissionManageGuild) {
		http.Error(w, "Missing permissions.", http.StatusUnauthorized)
		return
	}

	events, err := s.DB.Channels(discord.GuildID(resp.GetId()))
	if err != nil {
		common.Log.Errorf("Error getting events: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	for ev := range db.DefaultEventMap {
		id, err := strconv.ParseUint(r.FormValue(ev), 10, 64)
		if err != nil {
			common.Log.Infof("Invalid channel id `%v` for event %v", r.FormValue(ev), ev)
			id = 0
		}

		if id == 0 {
			events[ev] = discord.ChannelID(id)
		}

		isChannel := false
		for _, ch := range resp.GetChannels() {
			if ch.GetId() == id && (ch.GetType() == proto.GuildChannel_TEXT || ch.GetType() == proto.GuildChannel_NEWS) {
				isChannel = true
				break
			}
		}

		if !isChannel {
			id = 0
		}

		events[ev] = discord.ChannelID(id)
	}

	_, err = s.RPC.ClearCache(ctx, &proto.ClearCacheRequest{GuildId: resp.GetId()})
	if err != nil {
		common.Log.Errorf("Error clearing cache for %v: %v", resp.GetId(), err)
	}

	err = s.DB.SetChannels(discord.GuildID(resp.GetId()), events)
	if err != nil {
		http.Error(w, "Error setting channels.", http.StatusInternalServerError)
	}

	fmt.Fprint(w, "Success!")
	return
}
