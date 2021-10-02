package commands

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) redirectList(ctx bcr.Contexter) (err error) {
	conn, err := bot.DB.Obtain()
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	defer conn.Release()

	m, err := bot.DB.Redirects(conn, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if len(m) == 0 {
		return ctx.SendEphemeral("No channels are having their logs redirected.")
	}

	var s []string
	for k, v := range m {
		s = append(s, fmt.Sprintf("- <#%v> logging to <#%v>\n", k, v))
	}

	_, _, err = ctx.ButtonPages(
		bcr.StringPaginator("Channel log redirects", bcr.ColourPurple, s, 10), 15*time.Minute,
	)
	return err
}

func (bot *Bot) redirectTo(ctx bcr.Contexter) (err error) {
	conn, err := bot.DB.Obtain()
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	defer conn.Release()

	m, err := bot.DB.Redirects(conn, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	src, err := ctx.GetChannelFlag("from")
	if err != nil || (src.Type != discord.GuildText && src.Type != discord.GuildNews) || src.GuildID != ctx.GetGuild().ID {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	srcPerms, _ := ctx.Session().Permissions(src.ID, ctx.User().ID)
	if !srcPerms.Has(discord.PermissionViewChannel) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	dest, err := ctx.GetChannelFlag("to")
	if err != nil || (dest.Type != discord.GuildText && dest.Type != discord.GuildNews) || dest.GuildID != ctx.GetGuild().ID {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	destPerms, _ := ctx.Session().Permissions(dest.ID, ctx.User().ID)
	if !destPerms.Has(discord.PermissionViewChannel) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	m[src.ID.String()] = dest.ID

	err = bot.DB.SetRedirects(conn, ctx.GetGuild().ID, m)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendfX("Success! Events from %v will now log to %v.", src.Mention(), dest.Mention())
}

func (bot *Bot) redirectRemove(ctx bcr.Contexter) (err error) {
	conn, err := bot.DB.Obtain()
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	defer conn.Release()

	m, err := bot.DB.Redirects(conn, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	src, err := ctx.GetChannelFlag("from")
	if err != nil || (src.Type != discord.GuildText && src.Type != discord.GuildNews) || src.GuildID != ctx.GetGuild().ID {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	srcPerms, _ := ctx.Session().Permissions(src.ID, ctx.User().ID)
	if !srcPerms.Has(discord.PermissionViewChannel) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	delete(m, src.ID.String())

	err = bot.DB.SetRedirects(conn, ctx.GetGuild().ID, m)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendfX("Success! Events from %v will now log to the default log channel, if one is set.", src.Mention())
}
