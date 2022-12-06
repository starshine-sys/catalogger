package roles

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) roleUpdate(ev *gateway.GuildRoleUpdateEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// get previous version of role
	old, err := bot.Cabinet.Role(ctx, ev.GuildID, ev.Role.ID)
	if err != nil {
		log.Errorf("getting role %v in guild %v: %v", ev.Role.ID, ev.GuildID, err)
		return
	}

	// add new role version to cabinet when done
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := bot.Cabinet.SetRole(ctx, ev.GuildID, ev.Role)
		if err != nil {
			log.Errorf("setting role %v in %v: %v", ev.Role.ID, ev.GuildID, err)
		}
	}()

	if !bot.ShouldLog() {
		return
	}

	e := discord.Embed{
		Title: fmt.Sprintf(`Role "%v" updated`, ev.Role.Name),
		Color: common.ColourBlue,

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.Role.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	// only send an embed if something changed
	// (position changes send this event too, but we don't log those)
	var roleChanged bool

	// did the role's name change?
	if ev.Role.Name != old.Name {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Name",
			Value: fmt.Sprintf("**Before:** %v\n**After:** %v", old.Name, ev.Role.Name),
		})
		roleChanged = true
	}

	// did the role get hoisted/unhoisted? or did it become mentionable?
	if ev.Role.Hoist != old.Hoist || ev.Role.Mentionable != old.Mentionable {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "​",
			Value: fmt.Sprintf("**Mentionable:** %v\n**Shown separately:** %v", ev.Role.Mentionable, ev.Role.Hoist),
		})
		roleChanged = true
	}

	// did the role's colour change?
	if ev.Role.Color != old.Color {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Colour",
			Value: fmt.Sprintf("**Before:** %s\n**After:** %s", old.Color, ev.Role.Color),
		})
		roleChanged = true
	}

	// did the role's permissions change?
	if ev.Role.Permissions != old.Permissions {
		changedPerms := ev.Role.Permissions ^ old.Permissions

		var s string

		// perms added to the role
		if ev.Role.Permissions&changedPerms != 0 {
			s += fmt.Sprintf("+ %v", strings.Join(common.PermStrings(ev.Role.Permissions&changedPerms), ", "))
		}

		// spacer
		if ev.Role.Permissions&changedPerms != 0 && old.Permissions&changedPerms != 0 {
			s += "\n\n"
		}

		// perms removed from the role
		if old.Permissions&changedPerms != 0 {
			s += fmt.Sprintf("- %v", strings.Join(common.PermStrings(old.Permissions&changedPerms), ", "))
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Permissions",
			Value: "```diff\n" + s + "\n```",
		})
		roleChanged = true
	}

	if !roleChanged {
		return
	}

	bot.Send(ev.GuildID, ev, SendData{
		Embeds: []discord.Embed{e},
	})
}
