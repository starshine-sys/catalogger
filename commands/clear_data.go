package commands

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) clearData(ctx bcr.Contexter) (err error) {
	guildID := ctx.GetGuild().ID

	// get count
	var msgCount int64

	err = bot.DB.Pool.QueryRow(context.Background(), `select count(msg_id) from messages where server_id = $1`, guildID).Scan(&msgCount)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	yes, timeout := ctx.ConfirmButton(ctx.User().ID, bcr.ConfirmData{
		Message:   fmt.Sprintf("⚠️ **Are you sure you want to clear this server's data?** This will delete all logged messages (%v messages) and will clear your settings.", humanize.Comma(msgCount)),
		YesPrompt: "Delete data",
		YesStyle:  discord.DangerButton,
	})
	if timeout {
		_, err = send(ctx, "Operation timed out.")
		return
	}
	if !yes {
		_, err = send(ctx, "Operation cancelled.")
		return
	}

	c, err := bot.DB.Pool.Exec(context.Background(), "delete from messages where server_id = $1", guildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	deleted := c.RowsAffected()

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set channels = $1, ignored_channels = array[]::bigint[], banned_systems = array[]::char(5)[] where id = $2", db.DefaultEventMap, guildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	c, err = bot.DB.Pool.Exec(context.Background(), "delete from invites where guild_id = $1", guildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	invites := c.RowsAffected()

	c, err = bot.DB.Pool.Exec(context.Background(), "delete from watchlist where guild_id = $1", guildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	watchlist := c.RowsAffected()

	_, err = send(ctx, "Data deleted, %v messages, %v invites, and %v watchlist entries were deleted from the database.", deleted, invites, watchlist)
	return
}

func send(ctx bcr.Contexter, tmpl string, v ...interface{}) (*discord.Message, error) {
	return ctx.Session().SendMessageComplex(ctx.GetChannel().ID, api.SendMessageData{
		Content: fmt.Sprintf(tmpl, v...),
		AllowedMentions: &api.AllowedMentions{
			Parse: []api.AllowedMentionType{},
		},
	})
}
