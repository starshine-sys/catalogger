package commands

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/starshine-sys/bcr"
)

var requiredPerms = []bcr.Perm{
	{Permission: discord.PermissionManageGuild, Name: "Manage Server"},
	{Permission: discord.PermissionManageWebhooks, Name: "Manage Webhooks"},
	{Permission: discord.PermissionManageChannels, Name: "Manage Channels"},
	{Permission: discord.PermissionManageMessages, Name: "Manage Messages"},
	{Permission: discord.PermissionViewAuditLog, Name: "View Audit Log"},
	{Permission: discord.PermissionAttachFiles, Name: "Attach Files"},
	{Permission: discord.PermissionEmbedLinks, Name: "Embed Links"},
	{Permission: discord.PermissionAddReactions, Name: "Add Reactions"},
	{Permission: discord.PermissionSendMessages, Name: "Send Messages"},
	{Permission: discord.PermissionReadMessageHistory, Name: "Read Message History"},
	{Permission: discord.PermissionViewChannel, Name: "View Channel"},
}

func (bot *Bot) permcheck(ctx *bcr.Context) (err error) {
	ch, err := bot.DB.Channels(ctx.Message.GuildID)
	if err != nil {
		_, err = ctx.Sendf("Error getting channels: %v\nPlease contact the developer.", err)
		return
	}

	toCheck := map[discord.ChannelID]struct{}{}
	for _, v := range ch {
		if v.IsValid() {
			toCheck[v] = struct{}{}
		}
	}

	missingPerms := map[string][]string{}

	e := discord.Embed{
		Title: "Permission check",
		Fields: []discord.EmbedField{
			{
				Name: "Global permissions",
			},
		},
		Color: bcr.ColourGreen,
	}

	// global perms first
	perms, err := bot.globalPerms(ctx.Message.GuildID, ctx.Bot.ID)
	if err == nil {
		for _, p := range requiredPerms {
			if perms.Has(p.Permission) {
				e.Fields[0].Value += "✅"
			} else {
				e.Fields[0].Value += "❌"

				e.Color = bcr.ColourRed

				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Permission error",
					Value: "⚠️ The bot is missing one or more required permissions.\nIt may not work correctly or at all.",
				})
			}
			e.Fields[0].Value += fmt.Sprintf(" %v\n", p.Name)
		}
	}

	for ch := range toCheck {
		p, err := ctx.State.Permissions(ch, ctx.Bot.ID)
		if err != nil {
			_, err = ctx.Sendf("There was an error checking permissions for %v (ID: %v).", ch.Mention(), ch)
			return err
		}

		if !p.Has(discord.PermissionManageWebhooks) {
			missingPerms["webhooks"] = append(missingPerms["webhooks"], ch.Mention())
		}
		if !p.Has(discord.PermissionViewChannel) {
			missingPerms["view"] = append(missingPerms["view"], ch.Mention())
		}
	}

	if len(missingPerms["webhooks"]) == 0 && len(missingPerms["view"]) == 0 {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name: "Channel permissions",
			Value: fmt.Sprintf(`No issues found! :)
If the bot still isn't logging, try `+"`%vclearcache`"+`.
If that doesn't work, contact the developer.`, ctx.Prefix),
		})
	} else {
		e.Color = bcr.ColourRed

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Channel permissions",
			Value: "⚠️ The bot is missing required permissions in one or more channels.\nLogging may not work correctly or at all.",
		})

		if len(missingPerms["webhooks"]) > 0 {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Missing \"Manage Webhooks\"",
				Value: strings.Join(missingPerms["webhooks"], ", "),
			})
		}

		if len(missingPerms["view"]) > 0 {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Missing \"View Channel\"",
				Value: strings.Join(missingPerms["view"], ", "),
			})
		}
	}

	_, err = ctx.Send("", &e)
	return
}

func (bot *Bot) globalPerms(guildID discord.GuildID, userID discord.UserID) (perms discord.Permissions, err error) {
	// global perms first
	m, err := bot.State.Member(guildID, userID)
	if err != nil {
		return
	}

	g, err := bot.State.Session.Guild(guildID)
	if err != nil {
		return
	}

	for _, user := range m.RoleIDs {
		for _, r := range g.Roles {
			if user == r.ID {
				perms |= r.Permissions
			}
		}
	}

	return
}
