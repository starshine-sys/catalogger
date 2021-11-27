package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/web/proto"
)

func (s *server) delRedirect(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

	chID, err := strconv.ParseUint(params.ByName("channel"), 10, 64)
	if err != nil {
		http.Error(w, "Couldn't parse ID", http.StatusBadRequest)
		return
	}

	isChannel := false
	for _, ch := range resp.GetChannels() {
		if ch.GetId() == chID {
			isChannel = true
			break
		}
	}

	if !isChannel {
		http.Error(w, "Not a channel in this server.", http.StatusNotFound)
		return
	}

	m, err := s.DB.Redirects(discord.GuildID(resp.GetId()))
	if err != nil {
		common.Log.Errorf("Couldn't get current redirects: %v", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	delete(m, discord.ChannelID(chID).String())

	err = s.DB.SetRedirects(discord.GuildID(resp.GetId()), m)
	if err != nil {
		common.Log.Errorf("Couldn't set redirects: %v", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	_, err = s.RPC.ClearCache(ctx, &proto.ClearCacheRequest{GuildId: resp.GetId(), ChannelIds: []uint64{chID}})
	if err != nil {
		common.Log.Errorf("Error clearing cache for %v: %v", resp.GetId(), err)
	}

	fmt.Fprint(w, "Success!")
}
