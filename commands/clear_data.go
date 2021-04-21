package commands

import (
	"context"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) clearData(ctx *bcr.Context) (err error) {
	m, err := ctx.Send("⚠️ **Are you sure you want to clear this server's data?** This will delete all logged messages and will clear your settings.", nil)
	if err != nil {
		return err
	}

	yes, timeout := ctx.YesNoHandler(*m, ctx.Author.ID)
	if timeout {
		_, err = ctx.Send("Operation timed out.", nil)
		return
	}
	if !yes {
		_, err = ctx.Send("Operation cancelled.", nil)
		return
	}

	c, err := bot.DB.Pool.Exec(context.Background(), "delete from messages where server_id = $1", ctx.Message.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error deleting guild info for %v: %v", ctx.Message.GuildID, err)
		_, err = ctx.Send("There was an error deleting your data, please contact the bot owner for assistance.", nil)
		return
	}
	deleted := c.RowsAffected()

	c, err = bot.DB.Pool.Exec(context.Background(), "delete from pk_messages where server_id = $1", ctx.Message.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error deleting guild info for %v: %v", ctx.Message.GuildID, err)
		_, err = ctx.Send("There was an error deleting your data, please contact the bot owner for assistance.", nil)
		return
	}

	deleted += c.RowsAffected()

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set channels = $1, ignored_channels = array[]::bigint[], banned_systems = array[]::char(5)[] where id = $2", db.DefaultEventMap, ctx.Message.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error deleting guild info for %v: %v", ctx.Message.GuildID, err)
		_, err = ctx.Send("There was an error deleting your data, please contact the bot owner for assistance.", nil)
	}

	c, err = bot.DB.Pool.Exec(context.Background(), "delete from invites where guild_id = $1", ctx.Message.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error deleting guild info for %v: %v", ctx.Message.GuildID, err)
		_, err = ctx.Send("There was an error deleting your data, please contact the bot owner for assistance.", nil)
		return
	}

	_, err = ctx.Sendf("Data deleted, %v messages and %v invites were deleted from the database.", deleted, c.RowsAffected())
	return
}
