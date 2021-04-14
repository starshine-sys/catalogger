package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) guildRoleDelete(ev *gateway.GuildRoleDeleteEvent) {
	bot.RolesMu.Lock()
	old, ok := bot.Roles[ev.RoleID]
	delete(bot.Roles, ev.RoleID)
	if !ok {
		bot.RolesMu.Unlock()
		bot.Sugar.Errorf("Error getting info for role %v", ev.RoleID)
		return
	}
	bot.RolesMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["GUILD_ROLE_DELETE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("guild_role_delete", ev.GuildID, ch["GUILD_ROLE_DELETE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	e := discord.Embed{
		Title: fmt.Sprintf("Role \"%v\" deleted", old.Name),
		Description: fmt.Sprintf(`**Name:** %v
**Colour:** #%06X
**Mentionable:** %v
**Shown separately:** %v
**Position:** %v
Created %v`, old.Name, old.Color, old.Mentionable, old.Hoist, old.Position, bcr.HumanizeTime(bcr.DurationPrecisionSeconds, old.ID.Time())),

		Color: bcr.ColourRed,
		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.RoleID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	if old.Permissions != 0 {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Permissions",
			Value: strings.Join(bcr.PermStrings(old.Permissions), ", "),
		})
	}

	webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
}
