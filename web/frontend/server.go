package frontend

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-chi/chi/v5"
	"github.com/russross/blackfriday/v2"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
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
	From discord.Channel
	To   discord.Channel
}

type guild struct {
	Name    string
	ID      discord.GuildID
	IconURL string

	Channels []discord.Channel
}

func (s *server) rpcGuild(ctx context.Context, guildID discord.GuildID, client *userCache) (*proto.GuildResponse, error) {
	resp, err := s.RPC.Guild(ctx, &proto.GuildRequest{Id: uint64(guildID), UserId: uint64(client.User.ID)})
	if err == nil {
		if resp.Permissions == 0 {
			guilds, err := s.guilds(ctx, client)
			if err == nil {
				for _, g := range guilds {
					if g.ID == discord.GuildID(guildID) {
						resp.Permissions = uint64(g.Permissions)
					}
				}
			}
		}

		if discord.Permissions(resp.Permissions).Has(discord.PermissionAdministrator) {
			resp.Permissions = uint64(discord.PermissionAll)
		}
	}
	return resp, err
}

const guildNotFound = "You've taken a wrong turn! Either this isn't a server, or you're not in it, or you don't have permission to change its settings."

func (s *server) serverPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client := discordAPIFromSession(ctx)
	if client == nil {
		common.Log.Infof("Couldn't get a token from the request")
		loginRedirect(w, r)
		return
	}

	guildID, err := discord.ParseSnowflake(chi.URLParam(r, "id"))
	if err != nil {
		common.Log.Infof("Couldn't parse guild ID \"%v\"", chi.URLParam(r, "id"))
		s.error(w, http.StatusNotFound, false, guildNotFound)
		return
	}

	resp, err := s.rpcGuild(ctx, discord.GuildID(guildID), client)
	if err != nil || !discord.Permissions(resp.GetPermissions()).Has(discord.PermissionManageGuild) {
		if err == nil {
			common.Log.Infof("User %v has permissions %v, does not have permission to manage server.", client.User.Tag(), bcr.PermStrings(discord.Permissions(resp.GetPermissions())))
		} else {
			common.Log.Errorf("Error fetching guild: %v", err)
		}
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
		common.Log.Errorf("[%s] Error getting event channels: %v", id, err)
		return
	}

	// just gonna hack this for now, TODO: actually make this work well
	// basically doing this so *every* event has a channel set, even if that channel is 0
	for k := range db.DefaultEventMap {
		ch := data.CurrentChannels[k]
		data.CurrentChannels[k] = ch
	}

	sort.Slice(data.Redirects, func(i, j int) bool {
		return data.Redirects[i].From.Name < data.Redirects[j].From.Name
	})

	err = s.DB.QueryRow(ctx, "select ignored_channels from guilds where id = $1", resp.GetId()).Scan(&data.IgnoredChannels)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get this server's channels.")
		common.Log.Errorf("[%s] Error getting ignored channels: %v", id, err)
		return
	}

	for _, ch := range resp.GetChannels() {
		c := discord.Channel{
			ID:       discord.ChannelID(ch.GetId()),
			ParentID: discord.ChannelID(ch.GetParentID()),
			Name:     ch.GetName(),
			Position: int(ch.GetPosition()),
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
	data.Guild.Channels = common.SortChannels(data.Guild.Channels)

	redirMap, err := s.DB.Redirects(discord.GuildID(resp.GetId()))
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get this server's channels.")
		common.Log.Errorf("[%s] Error getting redirected channels: %v", id, err)
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
		common.Log.Errorf("[%s] Error getting guilds: %v", id, err)
		return
	}

	_, joined, unjoined, err := s.filterGuilds(ctx, guilds)
	if err != nil {
		id := s.error(w, http.StatusInternalServerError, true, "Couldn't get your servers.")
		common.Log.Errorf("[%s] Error filtering guilds: %v", id, err)
		return
	}

	data.JoinedGuilds = joined
	data.UnjoinedGuilds = unjoined

	err = tmpl.ExecuteTemplate(w, "server.html", data)
	if err != nil {
		common.Log.Errorf("Error executing template: %v", err)
		return
	}
}

const emojiBaseURL = "https://cdn.discordapp.com/emojis/"

var emojiMatch = regexp.MustCompile(`<(?P<animated>a)?:(?P<name>\w+):(?P<emoteID>\d{15,})>`)

var funcs template.FuncMap = map[string]interface{}{
	"selectOptions": func(channels []discord.Channel, selected discord.ChannelID) template.HTML {
		var (
			firstCategory = true
			hasCategory   = false
			b             strings.Builder
		)

		b.WriteString("<option value=\"0\">None</option>\n")

		for _, ch := range channels {
			if ch.Type == discord.GuildCategory {
				if !firstCategory {
					b.WriteString("</optgroup>\n")
				}
				b.WriteString(`<optgroup label="` + ch.Name + `">` + "\n")
				hasCategory = true

				continue
			}

			if ch.Type != discord.GuildText && ch.Type != discord.GuildNews {
				continue
			}

			b.WriteString("<option value=\"" + ch.ID.String() + "\"")
			if ch.ID == selected {
				b.WriteString(" selected")
			}
			b.WriteString(">#" + ch.Name + "</option>\n")
		}

		if hasCategory {
			b.WriteString("</optgroup>")
		}

		return template.HTML(strings.TrimSpace(b.String()))
	},
	"selectOptionsIgnoreMultiple": func(channels []discord.Channel, ignored []discord.ChannelID) template.HTML {
		var (
			firstCategory = true
			hasCategory   = false
			b             strings.Builder
		)

		for _, ch := range channels {
			if ch.Type == discord.GuildCategory {
				if !firstCategory {
					b.WriteString("</optgroup>\n")
				}
				b.WriteString(`<optgroup label="` + ch.Name + `">` + "\n")
				hasCategory = true

				continue
			}

			if ch.Type != discord.GuildText && ch.Type != discord.GuildNews {
				continue
			}

			if containsChannelID(ignored, ch.ID) {
				continue
			}

			b.WriteString(`<option value="` + ch.ID.String() + `">#` + ch.Name + "</option>\n")
		}

		if hasCategory {
			b.WriteString("</optgroup>")
		}

		return template.HTML(strings.TrimSpace(b.String()))
	},
	"multiselectOptions": func(channels []discord.Channel, selected []uint64) template.HTML {
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
	"markdownParse": func(s string) template.HTML {
		return template.HTML(blackfriday.Run(
			[]byte(s),
			blackfriday.WithExtensions(blackfriday.Autolink|blackfriday.Strikethrough|blackfriday.HardLineBreak)))
	},
	"emojiToImgs": func(s string) string {
		emojis := emojiMatch.FindAllString(s, -1)
		if emojis == nil {
			return s
		}

		for _, e := range emojis {
			ext := ".png"
			groups := emojiMatch.FindStringSubmatch(e)
			if groups[1] == "a" {
				ext = ".gif"
			}
			name := groups[2]
			url := emojiBaseURL + groups[3] + ext

			s = strings.NewReplacer(e, fmt.Sprintf(`<img class="emoji" src="%v" alt="%v" />`, url, name)).Replace(s)
		}

		return s
	},
	"timestamp": func(sf discord.MessageID) string {
		return sf.Time().Format("Jan 02, 15:04") + " UTC"
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
