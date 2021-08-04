package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) guildRoleCreate(ev *gateway.GuildRoleCreateEvent) {
	bot.RolesMu.Lock()
	bot.Roles[ev.Role.ID] = ev.Role
	bot.RolesMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "role_create",
			GuildID: ev.GuildID,
		}, err)
		return
	}
	if !ch["GUILD_ROLE_CREATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("guild_role_create", ev.GuildID, ch["GUILD_ROLE_CREATE"])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "role_create",
			GuildID: ev.GuildID,
		}, err)
		return
	}

	e := discord.Embed{
		Title: "Role created",
		Color: bcr.ColourGreen,

		Description: fmt.Sprintf(`**Name:** %v
**Colour:** #%06X
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

	bot.Queue(wh, "role_create", e)
}
