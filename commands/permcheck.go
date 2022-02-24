package commands

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
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

func (bot *Bot) permcheck(ctx bcr.Contexter) (err error) {
	ch, err := bot.DB.Channels(ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
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

	m, err := ctx.Session().Member(ctx.GetGuild().ID, bot.Router.Bot.ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	// global perms first
	perms, err := bot.globalPerms(ctx, m, ctx.GetGuild())
	if err == nil {
		for _, p := range requiredPerms {
			if perms.Has(p.Permission) {
				e.Fields[0].Value += "✅"
			} else {
				e.Fields[0].Value += "❌"

				e.Color = bcr.ColourRed

				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Permission error",
					Value: fmt.Sprintf("⚠️ %v is missing one or more required permissions.\nIt may not work correctly or at all.", bot.Router.Bot.Username),
				})
			}
			e.Fields[0].Value += fmt.Sprintf(" %v\n", p.Name)
		}
	}

	g := ctx.GetGuild()
	chs, err := ctx.Session().Channels(g.ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	for id := range toCheck {
		var ch discord.Channel
		for _, c := range chs {
			if c.ID == id {
				ch = c
				break
			}
		}

		p := discord.CalcOverwrites(*g, ch, *m)

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
If %v still isn't logging, try `+"`/clear-cache`"+`.
If that doesn't work, contact the developer.`, bot.Router.Bot.Username),
		})
	} else {
		e.Color = bcr.ColourRed

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Channel permissions",
			Value: fmt.Sprintf("⚠️ %v is missing required permissions in one or more channels.\nLogging may not work correctly or at all.", bot.Router.Bot.Username),
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

	_, err = ctx.Send("", e)
	return
}

func (bot *Bot) globalPerms(ctx bcr.Contexter, m *discord.Member, g *discord.Guild) (perms discord.Permissions, err error) {
	if m == nil || g == nil {
		return 0, nil
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
