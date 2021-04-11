package commands

import (
	"fmt"
	"os"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) help(ctx *bcr.Context) (err error) {
	// help for commands
	if len(ctx.Args) > 0 {
		return ctx.Help(ctx.Args)
	}

	e := &discord.Embed{
		Title: "Help",
		Description: fmt.Sprintf(`%v is a basic logging bot.
Right now, its main purpose is logging deleted messages proxied by PluralKit, and logging system info for new joins.`, ctx.Bot.Username),
		Color: bcr.ColourPurple,

		Fields: []discord.EmbedField{
			{
				Name:  "Info commands",
				Value: "`help`: show this help\n\n`ping`: show the bot's latency\n\n`invite`: get an invite link for the bot\n\n`events`: show all events (not even *close* to all of these are implemented; these are *all* Discord events)",
			},
			{
				Name:  "Configuration",
				Value: "`channels`: show which events are logging to which channels\n\n`setchannel`: log the given event to the current channel.\nFor example: `setchannel MESSAGE_DELETE`",
			},
			{
				Name:  "Available events",
				Value: "Events currently implemented:\n- `MESSAGE_DELETE`: deleted messages, both normal and PluralKit messages\n- `MESSAGE_UPDATE`: edited messages\n- `GUILD_MEMBER_ADD`: new member join",
			},
			{
				Name:  "Author",
				Value: "<@!694563574386786314> / starshine system 🌠✨#0001",
			},
			{
				Name:  "Source code",
				Value: "https://git.sr.ht/~starshine-sys/logger / [BSD 3-clause license](https://git.sr.ht/~starshine-sys/logger/tree/main/item/LICENSE)",
			},
		},
	}

	if os.Getenv("SUPPORT_SERVER") != "" {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Support",
			Value: fmt.Sprintf("Use this link to join the support server: %v", os.Getenv("SUPPORT_SERVER")),
		})
	}

	_, err = ctx.Send("", e)
	return
}

func (bot *Bot) invite(ctx *bcr.Context) (err error) {
	perms := discord.PermissionViewChannel |
		discord.PermissionReadMessageHistory |
		discord.PermissionAddReactions |
		discord.PermissionAttachFiles |
		discord.PermissionUseExternalEmojis |
		discord.PermissionEmbedLinks |
		discord.PermissionManageMessages |
		discord.PermissionSendMessages |
		discord.PermissionManageWebhooks |
		discord.PermissionManageGuild |
		discord.PermissionViewAuditLog

	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=%v&scope=bot%%20applications.commands", ctx.Bot.ID, perms)

	_, err = ctx.Sendf("Use the following link to invite me to your server: <%v>", link)
	return
}
