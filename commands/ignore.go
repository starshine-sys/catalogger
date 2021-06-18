package commands

import (
	"context"

	"github.com/starshine-sys/bcr"
)

func (bot *Bot) ignore(ctx *bcr.Context) (err error) {
	var blacklisted bool
	err = bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", ctx.Message.ChannelID, ctx.Message.GuildID).Scan(&blacklisted)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if blacklisted {
		_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_channels = array_remove(ignored_channels, $1) where id = $2", ctx.Channel.ID, ctx.Message.GuildID)
		if err != nil {
			return bot.DB.ReportCtx(ctx, err)
		}

		_, err = ctx.Send("Stopped ignoring this channel.", nil)
		return
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_channels = array_append(ignored_channels, $1) where id = $2", ctx.Channel.ID, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Send("Now ignoring this channel. Events here will not be logged.", nil)
	return
}
