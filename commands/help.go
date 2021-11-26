package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize/english"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) help(ctx bcr.Contexter) (err error) {
	// help for commands
	if v, ok := ctx.(*bcr.Context); ok {
		if len(v.Args) > 0 {
			return v.Help(v.Args)
		}
	}

	e := discord.Embed{
		Title: "Help",
		Description: fmt.Sprintf(`A logging bot that integrates with PluralKit's message proxying.
%v's prefixes are %v.
To get started, use `+"`%vsetchannel`"+` with one or more events.

[Basic usage guide](https://catalogger.starshines.xyz/docs) / [Privacy](https://catalogger.starshines.xyz/privacy)`, bot.Router.Bot.Username, english.OxfordWordSeries(bot.Router.Prefixes[:len(bot.Router.Prefixes)-1], "and"), bot.Router.Prefixes[0]),
		Color: bcr.ColourPurple,

		Fields: []discord.EmbedField{
			{
				Name:  "Info commands",
				Value: fmt.Sprintf("`help`: show this help\n`help permissions`: show a list of required permissions\n`help commands`: show a list of commands\n`stats`: show %v's latency and other stats\n`invite`: get an invite link for %v\n`events`: show all events", bot.Router.Bot.Username, bot.Router.Bot.Username),
			},
			{
				Name:  "Configuration",
				Value: fmt.Sprintf("`channels`: show which events are logging to which channels\n`setchannel`: log the given event(s) to the current channel\n**For example: `setchannel MESSAGE_DELETE, MESSAGE_UPDATE`**\n`ignorechannel`: ignore the current channel\n`redirect`: redirect a channel's logs to a different log channel\n`invites`: list this server's invites\n`invites name`: give an invite a name\n`watchlist`: show or configure this server's user watchlist.\n`cleardata`: clear this server's data (including messages)\n`clearcache`: clear %v's internal cache, in case logging is not working\n`permcheck`: check for permission errors in log channels", bot.Router.Bot.Username),
			},
			{
				Name:  "Events",
				Value: "For a list of events, see [here](https://github.com/starshine-sys/catalogger/blob/main/docs/USAGE.md#events).",
			},
			{
				Name:  "Author",
				Value: "<@!694563574386786314> / starshine#0001",
			},
			{
				Name:  "Source code",
				Value: "https://github.com/starshine-sys/catalogger / Licensed under the [GNU AGPLv3](https://github.com/starshine-sys/catalogger/blob/main/LICENSE)",
			},
		},
	}

	if os.Getenv("SUPPORT_SERVER") != "" {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Support",
			Value: fmt.Sprintf("Use this link to join the support server: %v", os.Getenv("SUPPORT_SERVER")),
		})
	}

	dashboard := os.Getenv("DASHBOARD_BASE")
	if dashboard != "" {
		e.Description += fmt.Sprintf("\n\nYou can also use the [dashboard](%v/servers) to configure %v!", dashboard, bot.Router.Bot.Username)
	}

	return ctx.SendX("", e)
}

func (bot *Bot) invite(ctx bcr.Contexter) (err error) {
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
		discord.PermissionManageChannels |
		discord.PermissionCreateInstantInvite

	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=%v&scope=bot%%20applications.commands", bot.Router.Bot.ID, perms)

	return ctx.SendEphemeral(fmt.Sprintf("Use the following link to invite %v to your server: <%v>", bot.Router.Bot.Username, link))
}

func (bot *Bot) perms(ctx bcr.Contexter) (err error) {
	e := discord.Embed{
		Title: "Permissions",
		Description: `This bot requires the following major permissions to function correctly:
- **Manage Webhooks**: to send log messages
- **Manage Server**: to track used invites
- **Manage Channels**: to track invite creation and deletion, and for more accurate used invite tracking`,
		Color: bcr.ColourPurple,
	}

	return ctx.SendEphemeral("", e)
}

func (bot *Bot) commands(ctx bcr.Contexter) (err error) {
	cmds := bot.Router.Commands()
	sort.Sort(bcr.Commands(cmds))

	var fields []discord.EmbedField
	for _, c := range cmds {
		v := c.Summary
		if v == "" {
			v = "No help given."
		}
		if c.Permissions != 0 {
			v += fmt.Sprintf("\nRequires the %v permission.", strings.Join(bcr.PermStrings(c.Permissions), ", "))
		}

		fields = append(fields, discord.EmbedField{
			Name:  c.Name,
			Value: v,
		})
	}

	_, _, err = ctx.ButtonPages(
		bcr.FieldPaginator("Commands", fmt.Sprintf("Use `%vhelp <name>` for more info on a command.", bot.Router.Prefixes[0]), bcr.ColourPurple, fields, 5), 15*time.Minute,
	)
	return
}

func (bot *Bot) dashboard(ctx bcr.Contexter) (err error) {
	dashboard := os.Getenv("DASHBOARD_BASE")
	if dashboard == "" {
		if _, ok := ctx.(*bcr.Context); ok {
			return
		}
		return ctx.SendEphemeral("There is no dashboard for this instance of the bot. Sorry :(")
	}

	perms, err := ctx.Session().Permissions(ctx.GetChannel().ID, ctx.User().ID)
	if err != nil {
		bot.Sugar.Errorf("Error fetching permissions for user: %v", err)
	}

	if !perms.Has(discord.PermissionManageGuild) || ctx.GetGuild() == nil {
		return ctx.SendEphemeral(fmt.Sprintf("%v's dashboard is available here: <%v/servers>", bot.Router.Bot.Username, dashboard))
	}

	return ctx.SendEphemeral(fmt.Sprintf("The dashboard for this server is available here: <%v/servers/%v>", dashboard, ctx.GetGuild().ID))
}
