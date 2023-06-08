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

	_, err = ctx.Reply("Added **%v** to the watchlist.", u.Tag())
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

	_, err = ctx.Reply("Removed **%v** from the watchlist.", u.Tag())
	return
}

func (bot *Bot) watchlistAddSlash(ctx bcr.Contexter) (err error) {
	u, err := ctx.GetUserFlag("user")
	if err != nil {
		return ctx.SendEphemeral("User not found.")
	}

	reason := ctx.GetStringFlag("reason")
	if reason == "" {
		reason = "N/A"
	}

	_, err = bot.DB.WatchlistAdd(ctx.GetGuild().ID, u.ID, ctx.User().ID, reason)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendfX("Added **%v** to the watchlist.", u.Tag())
}

func (bot *Bot) watchlistRemoveSlash(ctx bcr.Contexter) (err error) {
	u, err := ctx.GetUserFlag("user")
	if err != nil {
		return ctx.SendEphemeral("User not found.")
	}

	err = bot.DB.WatchlistRemove(ctx.GetGuild().ID, u.ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendfX("Removed **%v** from the watchlist.", u.Tag())
}
