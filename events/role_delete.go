package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/duration"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildRoleDelete(ev *gateway.GuildRoleDeleteEvent) (resp *handler.Response, err error) {
	old, ok := bot.Roles.Get(ev.RoleID)
	bot.Roles.Remove(ev.RoleID)
	if !ok {
		common.Log.Errorf("Error getting info for role %v", ev.RoleID)
		return
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildRoleDelete].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildRoleDelete],
		GuildID:   ev.GuildID,
	}

	e := discord.Embed{
		Title: fmt.Sprintf("Role \"%v\" deleted", old.Name),
		Description: fmt.Sprintf(`**Name:** %v
**Colour:** #%06X
**Mentionable:** %v
**Shown separately:** %v
**Position:** %v
Created <t:%v> (%v)`, old.Name, old.Color, old.Mentionable, old.Hoist, old.Position, old.ID.Time().Unix(), duration.FormatTime(old.ID.Time())),

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

	resp.Embeds = append(resp.Embeds, e)
	return
}
