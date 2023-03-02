package meta

import (
	"fmt"
	"net/url"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr/v2"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

const invitePerms = discord.PermissionViewChannel |
	discord.PermissionReadMessageHistory |
	discord.PermissionAttachFiles |
	discord.PermissionUseExternalEmojis |
	discord.PermissionEmbedLinks |
	discord.PermissionSendMessages |
	discord.PermissionManageWebhooks |
	discord.PermissionManageGuild |
	discord.PermissionViewAuditLog |
	discord.PermissionManageChannels |
	discord.PermissionCreateInstantInvite

func (bot *Bot) invite(ctx *bcr.CommandContext) (err error) {
	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%v&permissions=%v&scope=bot%%20applications.commands",
		bot.Me().ID, invitePerms)

	return ctx.ReplyEphemeral(fmt.Sprintf("Use this link to invite Catalogger to your server: <%v>", link))
}

func (bot *Bot) dashboard(ctx *bcr.CommandContext) (err error) {
	if bot.Config.Info.DashboardBase == "" {
		return ctx.ReplyEphemeral("This instance of Catalogger does not have the dashboard enabled, sorry :(")
	}

	if ctx.Guild == nil || ctx.Channel == nil || ctx.Member == nil {
		return ctx.ReplyEphemeral(fmt.Sprintf("Catalogger's dashboard is available here: <%v>", bot.Config.Info.DashboardBase))
	}

	perms := discord.CalcOverwrites(*ctx.Guild, *ctx.Channel, *ctx.Member)
	if !perms.Has(discord.PermissionManageGuild) {
		return ctx.ReplyEphemeral(fmt.Sprintf("Catalogger's dashboard is available here: <%v>", bot.Config.Info.DashboardBase))
	}

	dashPath, err := url.JoinPath(bot.Config.Info.DashboardBase, "/servers/", ctx.Event.GuildID.String())
	if err != nil {
		// this should never happen woops
		log.Errorf("building dashboard URL: %v", err)
		return bot.ReportError(ctx, err)
	}

	return ctx.ReplyEphemeral(fmt.Sprintf("The dashboard for %v is available here: <%v>", ctx.Guild.Name, dashPath))
}
