package commands

import (
	"strings"

	"github.com/starshine-sys/bcr"
)

func (bot *Bot) watchlistAdd(ctx *bcr.Context) (err error) {
	u, err := ctx.ParseUser(ctx.Args[0])
	if err != nil {
		_, err = ctx.Replyc(bcr.ColourRed, "User not found.")
		return
	}

	reason := "N/A"

	if len(ctx.Args) > 1 {
		reason = strings.TrimSpace(strings.TrimPrefix(ctx.RawArgs, ctx.Args[0]))
	}

	_, err = bot.DB.WatchlistAdd(ctx.Guild.ID, u.ID, ctx.Author.ID, reason)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Reply("Added **%v#%v** to the watchlist.", u.Username, u.Discriminator)
	return
}

func (bot *Bot) watchlistRemove(ctx *bcr.Context) (err error) {
	u, err := ctx.ParseUser(ctx.RawArgs)
	if err != nil {
		_, err = ctx.Replyc(bcr.ColourRed, "User not found.")
		return
	}

	err = bot.DB.WatchlistRemove(ctx.Guild.ID, u.ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Reply("Removed **%v#%v** from the watchlist.", u.Username, u.Discriminator)
	return
}
