package main

import (
	"fmt"
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
	ID         discord.ChannelID
	CategoryID discord.ChannelID
	Name       string
	Position   int32
	Type       discord.ChannelType
}

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
		fmt.Fprint(w, "Not a server.")
		return
	}

	resp, err := s.RPC.Guild(r.Context(), &proto.GuildRequest{Id: uint64(guildID), UserId: uint64(client.User.ID)})
	if err != nil {
		fmt.Fprint(w, "You're not in that server.")
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
		s.Sugar.Errorf("Error getting current event channels: %v", err)
		fmt.Fprintf(w, "Error getting current event channels: %v", err)
		return
	}

	sort.Slice(data.Redirects, func(i, j int) bool {
		return data.Redirects[i].From.Name < data.Redirects[j].From.Name
	})

	err = s.DB.Pool.QueryRow(ctx, "select ignored_channels from guilds where id = $1", resp.GetId()).Scan(&data.IgnoredChannels)
	if err != nil {
		s.Sugar.Errorf("Error getting current ignored channels: %v", err)
		fmt.Fprintf(w, "Error getting current ignored channels: %v", err)
		return
	}

	for _, ch := range resp.GetChannels() {
		c := channel{
			ID:         discord.ChannelID(ch.GetId()),
			CategoryID: discord.ChannelID(ch.GetCategoryId()),
			Name:       ch.GetName(),
			Position:   ch.GetPosition(),
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
		s.Sugar.Errorf("Error getting redirected channels: %v", err)
		fmt.Fprintf(w, "Error getting redirected channels: %v", err)
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
		s.Sugar.Errorf("Error getting guilds: %v", err)
		fmt.Fprintf(w, "Error getting guilds: %v", err)
		return
	}

	_, joined, unjoined, err := s.filterGuilds(ctx, guilds)
	if err != nil {
		s.Sugar.Errorf("Error filtering guilds: %v", err)
		fmt.Fprintf(w, "Error filtering guilds: %v", err)
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
