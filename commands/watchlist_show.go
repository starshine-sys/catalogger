package commands

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) watchlistSlash(ctx bcr.Contexter) (err error) {
	watchlist, err := bot.DB.GuildWatchlist(ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if len(watchlist) == 0 {
		return ctx.SendX("There are no users on the watchlist.")
	}

	fields := []discord.EmbedField{}

	for _, wl := range watchlist {
		fields = append(fields, bot.watchlistField(wl))
	}

	_, _, err = ctx.ButtonPages(bcr.FieldPaginator("Watchlist", "", bcr.ColourPurple, fields, 5), 15*time.Minute)
	return
}

func (bot *Bot) watchlist(ctx *bcr.Context) (err error) {
	watchlist, err := bot.DB.GuildWatchlist(ctx.Guild.ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if len(watchlist) == 0 {
		_, err = ctx.Reply("There are no users on the watchlist.")
		return
	}

	fields := []discord.EmbedField{}

	for _, wl := range watchlist {
		fields = append(fields, bot.watchlistField(wl))
	}

	_, _, err = ctx.ButtonPages(bcr.FieldPaginator("Watchlist", "", bcr.ColourPurple, fields, 5), 15*time.Minute)
	return
}

func (bot *Bot) watchlistField(wl db.WatchlistUser) (field discord.EmbedField) {
	var err error

	u, err := bot.User(wl.UserID)

	if err != nil {
		field.Name = "unknown user " + wl.UserID.String()
	} else {
		field.Name = u.Tag()
	}

	mod, err := bot.User(wl.Moderator)

	if err != nil {
		field.Value = fmt.Sprintf("**Moderator:** %v", wl.Moderator.Mention())
	} else {
		field.Value = fmt.Sprintf("**Moderator:** %v", mod.Tag())
	}

	field.Value += fmt.Sprintf("\n**Added:** <t:%v>\n\n**Reason:** ", wl.Added.Unix())

	for _, s := range wl.Reason {
		if len(field.Value) > 1020 {
			field.Value += "..."
			break
		}
		field.Value += string(s)
	}

	return field
}
