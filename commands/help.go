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
		Description: fmt.Sprintf(`A logging bot that integrates with PluralKit's message proxying.
The bot's prefix is `+"`%v`"+` (or a mention).
To get started, use `+"`%vsetchannel`"+` with one or more events (listed below).`, ctx.Prefix, ctx.Prefix),
		Color: bcr.ColourPurple,

		Fields: []discord.EmbedField{
			{
				Name:  "Info commands",
				Value: "`help`: show this help\n`help permissions`: show a list of required permissions\n`ping`: show the bot's latency\n`invite`: get an invite link for the bot\n`events`: show all events (not even *close* to all of these are implemented; these are *all* Discord events)",
			},
			{
				Name:  "Configuration",
				Value: "`channels`: show which events are logging to which channels\n`setchannel`: log the given event(s) to the current channel\n**For example: `setchannel MESSAGE_DELETE, MESSAGE_UPDATE`**\n`ignorechannel`: ignore the current channel\n`cleardata`: clear this server's data (including messages)\n`clearcache`: clear the bot's internal cache, in case logging is not working",
			},
			{
				Name:  "Available events",
				Value: "Events currently implemented:\n- `MESSAGE_DELETE`: deleted messages, both normal and PluralKit messages\n- `MESSAGE_UPDATE`: edited messages\n- `GUILD_MEMBER_ADD`: new member joining\n- `GUILD_MEMBER_REMOVE`: member leaving\n- `INVITE_CREATE`: created invites\n- `INVITE_DELETE`: deleted invites\n- `GUILD_BAN_ADD`: banned users\n- `GUILD_BAN_REMOVE`: unbanned users\n- `GUILD_MEMBER_UPDATE`: role updates\n- `GUILD_MEMBER_NICK_UPDATE`: username/nickname updates\n- `CHANNEL_CREATE`: channel creations\n- `CHANNEL_UPDATE`: channel updates\n- `CHANNEL_DELETE`: channel deletions",
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
		discord.PermissionViewAuditLog |
		discord.PermissionManageChannels

	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=%v&scope=bot%%20applications.commands", ctx.Bot.ID, perms)

	_, err = ctx.Sendf("Use the following link to invite me to your server: <%v>", link)
	return
}

func (bot *Bot) perms(ctx *bcr.Context) (err error) {
	e := discord.Embed{
		Title: "Permissions",
		Description: `This bot requires the following major permissions to function correctly:
- **Manage Webhooks**: to send log messages
- **Manage Server**: to track used invites
- **Manage Channels**: to track invite creation and deletion, and for more accurate used invite tracking`,
		Color: bcr.ColourPurple,
	}

	_, err = ctx.Send("", &e)
	return
}
