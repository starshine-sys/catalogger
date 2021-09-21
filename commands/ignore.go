package commands

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

const channelNotFoundError = "Channel not found, either it is not in this server, or it is not a text channel, or you do not have permission to view it."

func (bot *Bot) ignore(ctx bcr.Contexter) (err error) {
	guildID := ctx.GetGuild().ID
	var chID discord.ChannelID
	if v, ok := ctx.(*bcr.SlashContext); ok {
		ch, err := v.GetChannelFlag("channel")
		if err == nil && (ch.Type == discord.GuildNews || ch.Type == discord.GuildText) && ch.GuildID == ctx.GetGuild().ID {
			chID = ch.ID
		} else {
			return ctx.SendEphemeral(channelNotFoundError)
		}
	} else if v, ok := ctx.(*bcr.Context); ok {
		ch, err := v.ParseChannel(v.RawArgs)
		if err == nil && (ch.Type == discord.GuildNews || ch.Type == discord.GuildText) && ch.GuildID == ctx.GetGuild().ID {
			chID = ch.ID
		} else {
			return ctx.SendEphemeral(channelNotFoundError)
		}
	}

	if perms, _ := ctx.Session().Permissions(chID, ctx.User().ID); !perms.Has(discord.PermissionViewChannel) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	var blacklisted bool
	err = bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", chID, guildID).Scan(&blacklisted)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if blacklisted {
		_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_channels = array_remove(ignored_channels, $1) where id = $2", chID, guildID)
		if err != nil {
			return bot.DB.ReportCtx(ctx, err)
		}

		_, err = ctx.Sendf("Stopped ignoring %v.", chID.Mention())
		return
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_channels = array_append(ignored_channels, $1) where id = $2", chID, guildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Sendf("Now ignoring %v. Events there will not be logged.", chID.Mention())
	return
}
