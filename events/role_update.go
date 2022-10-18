package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildRoleUpdate(ev *gateway.GuildRoleUpdateEvent) (resp *handler.Response, err error) {
	old, ok := bot.Roles.Get(ev.Role.ID)
	if !ok {
		common.Log.Errorf("Error getting info for role %v", ev.Role.ID)
		return
	}
	bot.Roles.Set(ev.Role.ID, ev.Role)

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return nil, err
	}

	if !ch[keys.GuildRoleUpdate].IsValid() {
		return nil, nil
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildRoleUpdate],
		GuildID:   ev.GuildID,
	}

	e := discord.Embed{
		Title: fmt.Sprintf("Role \"%v\" updated", ev.Role.Name),
		Color: bcr.ColourBlue,

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.Role.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	// we need to filter out position changes
	var changed bool

	if ev.Role.Name != old.Name {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Name",
			Value: fmt.Sprintf("**Before:** %v\n**After:** %v", old.Name, ev.Role.Name),
		})
		changed = true
	}

	if ev.Role.Hoist != old.Hoist || ev.Role.Mentionable != old.Mentionable {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "​",
			Value: fmt.Sprintf("**Mentionable:** %v\n**Shown separately:** %v", ev.Role.Mentionable, ev.Role.Hoist),
		})
		changed = true
	}

	if ev.Role.Color != old.Color {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Colour",
			Value: fmt.Sprintf("**Before:** %s\n**After:** %s", old.Color, ev.Role.Color),
		})
		changed = true
	}

	if ev.Role.Permissions != old.Permissions {
		changedPerms := ev.Role.Permissions ^ old.Permissions

		var s string
		if ev.Role.Permissions&changedPerms != 0 {
			s += fmt.Sprintf("+ %v", strings.Join(bcr.PermStrings(ev.Role.Permissions&changedPerms), ", "))
		}
		if ev.Role.Permissions&changedPerms != 0 && old.Permissions&changedPerms != 0 {
			s += "\n\n"
		}
		if old.Permissions&changedPerms != 0 {
			s += fmt.Sprintf("- %v", strings.Join(bcr.PermStrings(old.Permissions&changedPerms), ", "))
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Permissions",
			Value: "```diff\n" + s + "\n```",
		})
		changed = true
	}

	if !changed {
		return nil, nil
	}

	resp.Embeds = append(resp.Embeds, e)
	return resp, nil
}
