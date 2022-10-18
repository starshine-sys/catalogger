package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildRoleCreate(ev *gateway.GuildRoleCreateEvent) (resp *handler.Response, err error) {
	bot.Roles.Set(ev.Role.ID, ev.Role)

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}
	if !ch[keys.GuildRoleCreate].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildRoleCreate],
		GuildID:   ev.GuildID,
	}

	e := discord.Embed{
		Title: "Role created",
		Color: bcr.ColourGreen,

		Description: fmt.Sprintf(`**Name:** %v
**Colour:** %s
**Mentionable:** %v
**Shown separately:** %v`, ev.Role.Name, ev.Role.Color, ev.Role.Mentionable, ev.Role.Hoist),

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.Role.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	if ev.Role.Permissions != 0 {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Permissions",
			Value: strings.Join(bcr.PermStrings(ev.Role.Permissions), "\n"),
		})
	}

	resp.Embeds = append(resp.Embeds, e)
	return
}
