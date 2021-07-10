package commands

import (
	"context"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) renameInvite(ctx *bcr.Context) (err error) {
	is, err := ctx.State.GuildInvites(ctx.Message.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting guild invites: %v", err)
		_, err = ctx.Sendf("Could not get this server's invites. Are you sure I have the **Manage Server** permission?")
		return
	}

	var (
		inv   discord.Invite
		found bool
	)
	for _, i := range is {
		if i.Code == ctx.Args[0] {
			inv = i
			found = true
			break
		}
	}

	if !found {
		_, err = ctx.Sendf("Could not find an invite with that name (``%v``).", bcr.EscapeBackticks(ctx.Args[0]))
		return
	}

	name := strings.TrimSpace(strings.TrimPrefix(ctx.RawArgs, ctx.Args[0]))

	if strings.EqualFold(name, "-clear") {
		_, err = bot.DB.Pool.Exec(context.Background(), "delete from invites where code = $1", inv.Code)
	} else {
		err = bot.DB.NameInvite(ctx.Message.GuildID, inv.Code, name)
	}
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if strings.EqualFold(name, "-clear") {
		_, err = ctx.Send("Invite name reset!")
		return
	}
	_, err = ctx.Send("Invite name set!")
	return
}
