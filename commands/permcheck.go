package commands

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
)

var requiredGlobalPerms = []bcr.Perm{
	{Permission: discord.PermissionManageGuild, Name: "Manage Server"},
	{Permission: discord.PermissionViewAuditLog, Name: "View Audit Log"},
}

var requiredChannelPerms = []bcr.Perm{
	{Permission: discord.PermissionManageChannels, Name: "Manage Channel"},
	{Permission: discord.PermissionAttachFiles, Name: "Attach Files"},
	{Permission: discord.PermissionEmbedLinks, Name: "Embed Links"},
	{Permission: discord.PermissionAddReactions, Name: "Add Reactions"},
	{Permission: discord.PermissionSendMessages, Name: "Send Messages"},
	{Permission: discord.PermissionReadMessageHistory, Name: "Read Message History"},
	{Permission: discord.PermissionViewChannel, Name: "View Channel"},
}

var manageWebhooks = bcr.Perm{Permission: discord.PermissionManageWebhooks, Name: "Manage Webhooks"}

func (bot *Bot) permcheck(ctx bcr.Contexter) (err error) {
	ch, err := bot.DB.Channels(ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	var (
		needWebhookPerms = make(map[discord.ChannelID]struct{})
		missingPerms     = make(map[discord.Permissions][]string)
		ignored          int
		channelIssues    bool
	)

	for _, v := range ch {
		if v.IsValid() {
			needWebhookPerms[v] = struct{}{}
		}
	}

	e := discord.Embed{
		Title: "Permission check",
		Fields: []discord.EmbedField{
			{
				Name: "Server permissions",
			},
		},
		Color: bcr.ColourGreen,
	}

	m, err := ctx.Session().Member(ctx.GetGuild().ID, bot.Router.Bot.ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	user := ctx.GetMember()

	// global perms first
	perms := bot.globalPerms(ctx, m, ctx.GetGuild())
	if perms != 0 {
		for _, p := range requiredGlobalPerms {
			if perms.Has(p.Permission) {
				e.Fields[0].Value += "✅"
			} else {
				e.Fields[0].Value += "❌"

				e.Color = bcr.ColourRed

				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Permission error",
					Value: fmt.Sprintf("⚠️ %v is missing one or more required server permissions.\nIt may not work correctly or at all.", bot.Router.Bot.Username),
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
	// sort channels
	chs = common.SortChannels(chs)

	for _, ch := range chs {
		if ch.Type != discord.GuildNews && ch.Type != discord.GuildText {
			continue
		}

		// ignore channels that the user can't see
		if !discord.CalcOverwrites(*g, ch, *user).Has(discord.PermissionViewChannel) {
			ignored++
			continue
		}

		p := discord.CalcOverwrites(*g, ch, *m)
		if _, ok := needWebhookPerms[ch.ID]; ok {
			// logging channel, also needs manage webhooks
			if !p.Has(discord.PermissionManageWebhooks) {
				channelIssues = true
				missingPerms[discord.PermissionManageWebhooks] = append(missingPerms[discord.PermissionManageWebhooks], ch.Mention())
			}
		}

		for _, perm := range requiredChannelPerms {
			if !p.Has(perm.Permission) {
				channelIssues = true
				missingPerms[perm.Permission] = append(missingPerms[perm.Permission], ch.Mention())
			}
		}
	}

	if channelIssues {
		e.Color = bcr.ColourRed
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Channel permissions",
			Value: fmt.Sprintf("⚠️ %v is missing required permissions in one or more channels.\nLogging may not work correctly or at all.", bot.Router.Bot.Username),
		})

		// range over all permissions
		for _, perm := range append([]bcr.Perm{manageWebhooks}, requiredChannelPerms...) {
			if len(missingPerms[perm.Permission]) > 0 {
				var val strings.Builder
				for i, ch := range missingPerms[perm.Permission] {
					if val.Len()+len(ch) > 1000 {
						val.WriteString(fmt.Sprintf("and %v more", len(missingPerms[perm.Permission])-i))
						break
					}

					val.WriteString(ch)

					if i != len(missingPerms[perm.Permission])-1 {
						val.WriteString(", ")
					}
				}

				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  fmt.Sprintf(`Missing "%v"`, perm.Name),
					Value: val.String(),
				})
			}
		}

	} else {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name: "Channel permissions",
			Value: fmt.Sprintf(`No issues found! :)
If %v still isn't logging, try `+"`/clear-cache`"+`.
If that doesn't work, contact the developer.`, bot.Router.Bot.Username),
		})
	}

	if ignored != 0 {
		if ignored == 1 {
			e.Footer = &discord.EmbedFooter{
				Text: "1 channel was ignored as you do not have view access to it.",
			}
		} else {
			e.Footer = &discord.EmbedFooter{
				Text: fmt.Sprintf("%v channels were ignored as you do not have view access to them.", ignored),
			}
		}
	}

	_, err = ctx.Send("", e)
	return
}

func (bot *Bot) globalPerms(ctx bcr.Contexter, m *discord.Member, g *discord.Guild) (perms discord.Permissions) {
	if m == nil || g == nil {
		return 0
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
