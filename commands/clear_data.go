package commands

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) clearData(ctx *bcr.Context) (err error) {
	// get count
	var msgCount int64

	err = bot.DB.Pool.QueryRow(context.Background(), `select count(msg_id) from messages where server_id = $1`, ctx.Message.GuildID).Scan(&msgCount)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	yes, timeout := ctx.ConfirmButton(ctx.Author.ID, bcr.ConfirmData{
		Message:   fmt.Sprintf("⚠️ **Are you sure you want to clear this server's data?** This will delete all logged messages (%v messages) and will clear your settings.", humanize.Comma(msgCount)),
		YesPrompt: "Delete data",
		YesStyle:  discord.DangerButton,
	})
	if timeout {
		_, err = ctx.Send("Operation timed out.")
		return
	}
	if !yes {
		_, err = ctx.Send("Operation cancelled.")
		return
	}

	c, err := bot.DB.Pool.Exec(context.Background(), "delete from messages where server_id = $1", ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	deleted := c.RowsAffected()

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set channels = $1, ignored_channels = array[]::bigint[], banned_systems = array[]::char(5)[] where id = $2", db.DefaultEventMap, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	c, err = bot.DB.Pool.Exec(context.Background(), "delete from invites where guild_id = $1", ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Sendf("Data deleted, %v messages and %v invites were deleted from the database.", deleted, c.RowsAffected())
	return
}
