package main

import (
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/julienschmidt/httprouter"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/web/proto"
)

type serverPageData struct {
	User  *discord.User
	Guild guild

	JoinedGuilds   []guildList
	UnjoinedGuilds []guildList

	CurrentChannels db.EventMap

	IgnoredChannels []uint64

	RedirectedChannels []discord.ChannelID
	Redirects          []redirect
}

type redirect struct {
	From channel
	To   channel
}

type guild struct {
	Name    string
	ID      discord.GuildID
	IconURL string

	Channels []channel
}

type channel struct {
	ID       discord.ChannelID
	ParentID discord.ChannelID
	Name     string
	Position int32
	Type     discord.ChannelType
}

const guildNotFound = "You've taken a wrong turn! Either this isn't a server, or you're not in it, or you don't have permission to change its settings."

func (s *server) serverPage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := r.Context()

	client := discordAPIFromSession(ctx)
	if client == nil {
		s.Sugar.Infof("Couldn't get a token from the request")
		loginRedirect(w, r)
		return
	}

	guildID, err := discord.ParseSnowflake(params.ByName("id"))
	if err != nil {
		s.error(w, http.StatusNotFound, false, guildNotFound)
		return
	}

	resp, err := s.RPC.Guild(r.Context(), &proto.GuildRequest{Id: uint64(guildID), UserId: uint64(client.User.ID)})
	if err != nil || !discord.Permissions(resp.GetPermissions()).Has(discord.PermissionManageGuild) {
		s.error(w, http.StatusNotFound, false, guildNotFound)
		return
	}

	data := serverPageData{
		User: client.User,
		Guild: guild{
			Name:    resp.GetName(),
			ID:      discord.GuildID(resp.GetId()),
			IconURL: resp.GetIcon(),
		},
	}

	data.CurrentChannels, err = s.DB.Channels(discord.GuildID(resp.GetId()))
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get this server's channels.")
		s.Sugar.Errorf("[%s] Error getting event channels: %v", id, err)
		return
	}

	sort.Slice(data.Redirects, func(i, j int) bool {
		return data.Redirects[i].From.Name < data.Redirects[j].From.Name
	})

	err = s.DB.QueryRow(ctx, "select ignored_channels from guilds where id = $1", resp.GetId()).Scan(&data.IgnoredChannels)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get this server's channels.")
		s.Sugar.Errorf("[%s] Error getting ignored channels: %v", id, err)
		return
	}

	for _, ch := range resp.GetChannels() {
		c := channel{
			ID:       discord.ChannelID(ch.GetId()),
			ParentID: discord.ChannelID(ch.GetParentID()),
			Name:     ch.GetName(),
			Position: ch.GetPosition(),
		}

		switch ch.GetType() {
		case proto.GuildChannel_UNKNOWN:
			// just use a type that we Cannot support
			c.Type = discord.GuildStore
		case proto.GuildChannel_TEXT:
			c.Type = discord.GuildText
		case proto.GuildChannel_NEWS:
			c.Type = discord.GuildNews
		case proto.GuildChannel_CATEGORY:
			c.Type = discord.GuildCategory
		case proto.GuildChannel_VOICE:
			c.Type = discord.GuildVoice
		case proto.GuildChannel_THREAD:
			c.Type = discord.GuildPublicThread
		}

		data.Guild.Channels = append(data.Guild.Channels, c)
	}

	sort.Slice(data.Guild.Channels, func(i, j int) bool {
		return data.Guild.Channels[i].Name < data.Guild.Channels[j].Name
	})

	redirMap, err := s.DB.Redirects(discord.GuildID(resp.GetId()))
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get this server's channels.")
		s.Sugar.Errorf("[%s] Error getting redirected channels: %v", id, err)
		return
	}

	for k, v := range redirMap {
		for _, ch := range data.Guild.Channels {
			if k == ch.ID.String() {
				r := redirect{From: ch}

				matched := false
				for _, ch := range data.Guild.Channels {
					if ch.ID == v {
						matched = true
						r.To = ch
						break
					}
				}

				if matched {
					data.Redirects = append(data.Redirects, r)
					data.RedirectedChannels = append(data.RedirectedChannels, ch.ID)
				}
			}
		}
	}

	guilds, err := s.guilds(ctx, client)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get your servers.")
		s.Sugar.Errorf("[%s] Error getting guilds: %v", id, err)
		return
	}

	_, joined, unjoined, err := s.filterGuilds(ctx, guilds)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get your servers.")
		s.Sugar.Errorf("[%s] Error filtering guilds: %v", id, err)
		return
	}

	data.JoinedGuilds = joined
	data.UnjoinedGuilds = unjoined

	err = tmpl.ExecuteTemplate(w, "server.html", data)
	if err != nil {
		s.Sugar.Errorf("Error executing template: %v", err)
		return
	}
}

var funcs template.FuncMap = map[string]interface{}{
	"selectOptions": func(channels []channel, selected discord.ChannelID) template.HTML {
		var b strings.Builder

		b.WriteString("<option value=\"0\">None</option>\n")

		for _, ch := range channels {
			if ch.Type != discord.GuildText {
				continue
			}

			b.WriteString("<option value=\"" + ch.ID.String() + "\"")
			if ch.ID == selected {
				b.WriteString(" selected")
			}
			b.WriteString(">#" + ch.Name + "</option>\n")
		}
		return template.HTML(strings.TrimSpace(b.String()))
	},
	"selectOptionsIgnoreMultiple": func(channels []channel, ignored []discord.ChannelID) template.HTML {
		var b strings.Builder

		for _, ch := range channels {
			if ch.Type != discord.GuildText && ch.Type != discord.GuildNews {
				continue
			}

			if containsChannelID(ignored, ch.ID) {
				continue
			}

			b.WriteString(`<option value="` + ch.ID.String() + `">#` + ch.Name + "</option>\n")
		}

		return template.HTML(strings.TrimSpace(b.String()))
	},
	"multiselectOptions": func(channels []channel, selected []uint64) template.HTML {
		var b strings.Builder

		for _, ch := range channels {
			if ch.Type != discord.GuildText && ch.Type != discord.GuildNews {
				continue
			}

			b.WriteString(`<option value="` + ch.ID.String() + `"`)
			if containsID(selected, ch.ID) {
				b.WriteString(" selected")
			}
			b.WriteString(">#" + ch.Name + "</option>\n")
		}
		return template.HTML(strings.TrimSpace(b.String()))
	},
}

func containsID(list []uint64, id discord.ChannelID) bool {
	for _, i := range list {
		if discord.ChannelID(i) == id {
			return true
		}
	}
	return false
}

func containsChannelID(list []discord.ChannelID, id discord.ChannelID) bool {
	for _, i := range list {
		if i == id {
			return true
		}
	}
	return false
}
