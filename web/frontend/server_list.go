package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/web/proto"
)

var tmpl = template.Must(template.New("servers").Parse(tmplData))

type guildList struct {
	Name string
	Link string
}

func (s *server) serverList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client := DiscordAPIFromSession(r.Context())
	if client == nil {
		s.Sugar.Infof("Couldn't get a token from the request")
		loginRedirect(w, r)
		return
	}

	u, err := client.Me()
	if err != nil {
		s.Sugar.Errorf("Error getting user: %v", err)
		fmt.Fprintf(w, "Error getting user: %v", err)
		return
	}

	guilds, err := client.Guilds(200)
	if err != nil {
		s.Sugar.Errorf("Error getting guilds: %v", err)
		fmt.Fprintf(w, "Error getting guilds: %v", err)
		return
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
	}{User: u}

	resp, err := s.RPC.UserGuildList(r.Context(), &proto.UserGuildListRequest{GuildId: guildIDs})
	if err != nil {
		s.Sugar.Errorf("Error filtering guilds: %v", err)
		fmt.Fprintf(w, "Error filtering guilds: %v", err)
		return
	}

	for _, g := range resp.GetGuilds() {
		for _, dg := range guilds {
			if g.GetId() == uint64(dg.ID) {
				name := dg.Name
				link := "/servers/" + dg.ID.String()
				if !g.GetJoined() {
					name = dg.Name + " [click to invite]"
					link = inviteLink(dg.ID)
				}

				data.Guilds = append(data.Guilds, guildList{
					Name: name,
					Link: link,
				})

				break
			}
		}
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		s.Sugar.Errorf("Error executing template: %v", err)
		return
	}
}

func inviteLink(guildID discord.GuildID) string {
	return fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=537259248&scope=bot%%20applications.commands&guild_id=%v&disable_guild_select=true", clientID, guildID)
}

var tmplData string = `<!DOCTYPE html>
<html>
<head>
	<title>Catalogger</title>
</head>
<body>
	<h1>Hi {{.User.Username}}#{{.User.Discriminator}}</h1>

	<ul>
	{{range .Guilds}}
		<li><a href="{{.Link}}">{{.Name}}</li>
	{{end}}
	</ul>
</body>
</html>`
