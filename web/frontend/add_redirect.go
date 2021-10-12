package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/web/proto"
)

func (s *server) addRedirect(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

	fromID, err := strconv.ParseUint(r.FormValue("from-channel"), 10, 64)
	if err != nil {
		http.Error(w, "Couldn't parse ID", http.StatusBadRequest)
		return
	}

	toID, err := strconv.ParseUint(r.FormValue("to-channel"), 10, 64)
	if err != nil {
		http.Error(w, "Couldn't parse ID", http.StatusBadRequest)
		return
	}

	var fromChannel, toChannel *discord.Channel

	for _, ch := range resp.GetChannels() {
		if ch.GetId() == fromID {
			fromChannel = &discord.Channel{
				ID:   discord.ChannelID(ch.GetId()),
				Name: ch.GetName(),
			}
		}
		if ch.GetId() == toID && (ch.GetType() == proto.GuildChannel_TEXT || ch.GetType() == proto.GuildChannel_NEWS) {
			toChannel = &discord.Channel{
				ID:   discord.ChannelID(ch.GetId()),
				Name: ch.GetName(),
			}
		}
	}

	if fromChannel == nil || toChannel == nil {
		http.Error(w, "Channel(s) not found", http.StatusBadRequest)
		return
	}

	from := discord.ChannelID(fromID)
	to := discord.ChannelID(toID)

	m, err := s.DB.Redirects(discord.GuildID(resp.GetId()))
	if err != nil {
		s.Sugar.Errorf("Couldn't get current redirects: %v", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	m[from.String()] = to

	err = s.DB.SetRedirects(discord.GuildID(resp.GetId()), m)
	if err != nil {
		s.Sugar.Errorf("Couldn't set redirects: %v", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	_, err = s.RPC.ClearCache(ctx, &proto.ClearCacheRequest{GuildId: resp.GetId(), ChannelIds: []uint64{fromID, toID}})
	if err != nil {
		s.Sugar.Errorf("Error clearing cache for %v: %v", resp.GetId(), err)
	}

	type ch struct {
		ID   discord.ChannelID `json:"id"`
		Name string            `json:"name"`
	}

	data := struct {
		GuildID discord.GuildID `json:"guild_id"`
		From    ch              `json:"from"`
		To      ch              `json:"to"`
	}{
		GuildID: discord.GuildID(resp.GetId()),
		From: ch{
			ID:   fromChannel.ID,
			Name: fromChannel.Name,
		},
		To: ch{
			ID:   toChannel.ID,
			Name: toChannel.Name,
		},
	}

	b, err := json.Marshal(&data)
	if err != nil {
		s.Sugar.Errorf("Error marshaling JSON: %v", err)
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return
}
