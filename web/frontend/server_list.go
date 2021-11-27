package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/web/proto"
)

//go:embed templates
var fs embed.FS

var tmpl = template.Must(template.New("servers").Funcs(funcs).ParseFS(fs, "templates/*"))

type guildList struct {
	ID      discord.GuildID
	Name    string
	Link    string
	Joined  bool
	IconURL string
}

func (s *server) guilds(ctx context.Context, client *userCache) (guilds []discord.Guild, err error) {
	guilds, err = s.guildList(ctx, client.User.ID)
	if err != nil {
		guilds, err = client.Guilds(200)
		if err != nil {
			return
		}
		err = s.setGuildList(ctx, client.User.ID, guilds)
		if err != nil {
			return
		}
	}
	return guilds, err
}

func (s *server) serverList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	client := discordAPIFromSession(r.Context())
	if client == nil {
		common.Log.Infof("Couldn't get a token from the request")
		loginRedirect(w, r)
		return
	}

	guilds, err := s.guilds(ctx, client)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get your servers.")
		common.Log.Errorf("[%s] Error getting guilds: %v", id, err)
		return
	}

	filtered, joined, unjoined, err := s.filterGuilds(ctx, guilds)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get your servers.")
		common.Log.Errorf("[%s] Error filtering guilds: %v", id, err)
		return
	}

	data := struct {
		User   *discord.User
		Guilds []guildList

		JoinedGuilds   []guildList
		UnjoinedGuilds []guildList
	}{
		User:           client.User,
		Guilds:         filtered,
		JoinedGuilds:   joined,
		UnjoinedGuilds: unjoined,
	}

	err = tmpl.ExecuteTemplate(w, "server_list.html", data)
	if err != nil {
		common.Log.Errorf("Error executing template: %v", err)
		return
	}
}

func inviteLink(guildID discord.GuildID) string {
	return fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=537259249&scope=bot%%20applications.commands&guild_id=%v&disable_guild_select=true", clientID, guildID)
}

func (s *server) guildList(ctx context.Context, id discord.UserID) (list []discord.Guild, err error) {
	b := []byte{}

	err = s.Redis.Do(ctx, radix.Cmd(&b, "GET", "user-guilds:"+id.String()))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (s *server) setGuildList(ctx context.Context, id discord.UserID, guilds []discord.Guild) error {
	b, err := json.Marshal(guilds)
	if err != nil {
		return err
	}

	// TODO: change timeout
	err = s.Redis.Do(ctx, radix.Cmd(nil, "SET", "user-guilds:"+id.String(), string(b), "EX", "3600"))
	return err
}

func (s *server) filterGuilds(ctx context.Context, gs []discord.Guild) (guilds []guildList, joinedGuilds []guildList, unjoinedGuilds []guildList, err error) {
	old := gs
	gs = nil
	guildIDs := []uint64{}
	for _, g := range old {
		if g.Owner || g.Permissions.Has(discord.PermissionAdministrator) || g.Permissions.Has(discord.PermissionManageGuild) {
			guildIDs = append(guildIDs, uint64(g.ID))
			gs = append(gs, g)
		}
	}

	resp, err := s.RPC.UserGuildList(ctx, &proto.UserGuildListRequest{GuildIds: guildIDs})
	if err != nil {
		return nil, nil, nil, err
	}

	for _, g := range resp.GetGuilds() {
		for _, dg := range gs {
			if g.GetId() == uint64(dg.ID) {
				link := "/servers/" + dg.ID.String()
				if !g.GetJoined() {
					link = inviteLink(dg.ID)
				}

				icon := dg.IconURLWithType(discord.PNGImage)
				if icon == "" {
					icon = "https://cdn.discordapp.com/embed/avatars/1.png"
				}

				gl := guildList{
					ID:      dg.ID,
					Name:    dg.Name,
					Link:    link,
					Joined:  g.GetJoined(),
					IconURL: icon + "?size=128",
				}

				guilds = append(guilds, gl)

				if g.GetJoined() {
					joinedGuilds = append(joinedGuilds, gl)
				} else {
					unjoinedGuilds = append(unjoinedGuilds, gl)
				}

				break
			}
		}
	}

	return
}
