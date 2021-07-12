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
	"github.com/starshine-sys/catalogger/web/proto"
)

//go:embed templates
var fs embed.FS

var tmpl = template.Must(template.New("servers").ParseFS(fs, "templates/*"))

type guildList struct {
	Name    string
	Link    string
	Joined  bool
	IconURL string
}

func (s *server) serverList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	client := discordAPIFromSession(r.Context())
	if client == nil {
		s.Sugar.Infof("Couldn't get a token from the request")
		loginRedirect(w, r)
		return
	}

	guilds, err := s.guildList(ctx, client.User.ID)
	if err != nil {
		guilds, err = client.Guilds(200)
		if err != nil {
			s.Sugar.Errorf("Error getting guilds: %v", err)
			fmt.Fprintf(w, "Error getting guilds: %v", err)
			return
		}
		err = s.setGuildList(ctx, client.User.ID, guilds)
		if err != nil {
			s.Sugar.Errorf("Error setting guild list: %v", err)
			fmt.Fprintf(w, "Error setting guild list: %v", err)
			return
		}
	}

	old := guilds
	guilds = nil
	guildIDs := []uint64{}
	for _, g := range old {
		if g.Owner || g.Permissions.Has(discord.PermissionAdministrator) || g.Permissions.Has(discord.PermissionManageGuild) {
			guildIDs = append(guildIDs, uint64(g.ID))
			guilds = append(guilds, g)
		}
	}

	data := struct {
		User   *discord.User
		Guilds []guildList
	}{User: client.User}

	resp, err := s.RPC.UserGuildList(r.Context(), &proto.UserGuildListRequest{GuildIds: guildIDs})
	if err != nil {
		s.Sugar.Errorf("Error filtering guilds: %v", err)
		fmt.Fprintf(w, "Error filtering guilds: %v", err)
		return
	}

	for _, g := range resp.GetGuilds() {
		for _, dg := range guilds {
			if g.GetId() == uint64(dg.ID) {
				link := "/servers/" + dg.ID.String()
				if !g.GetJoined() {
					link = inviteLink(dg.ID)
				}

				icon := dg.IconURLWithType(discord.PNGImage)
				if icon == "" {
					icon = "https://cdn.discordapp.com/embed/avatars/1.png"
				}

				data.Guilds = append(data.Guilds, guildList{
					Name:    dg.Name,
					Link:    link,
					Joined:  g.GetJoined(),
					IconURL: icon + "?size=128",
				})

				break
			}
		}
	}

	err = tmpl.ExecuteTemplate(w, "server_list.html", data)
	if err != nil {
		s.Sugar.Errorf("Error executing template: %v", err)
		return
	}
}

func inviteLink(guildID discord.GuildID) string {
	return fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=537259248&scope=bot%%20applications.commands&guild_id=%v&disable_guild_select=true", clientID, guildID)
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
