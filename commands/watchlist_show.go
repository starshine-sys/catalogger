package commands

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
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

	m := map[discord.UserID]*discord.User{}

	fields := []discord.EmbedField{}

	for _, wl := range watchlist {
		fields = append(fields, bot.watchlistField(ctx.Session(), m, wl))
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

	m := map[discord.UserID]*discord.User{}

	fields := []discord.EmbedField{}

	for _, wl := range watchlist {
		fields = append(fields, bot.watchlistField(ctx.State, m, wl))
	}

	_, _, err = ctx.ButtonPages(bcr.FieldPaginator("Watchlist", "", bcr.ColourPurple, fields, 5), 15*time.Minute)
	return
}

func (bot *Bot) watchlistField(s *state.State, m map[discord.UserID]*discord.User, wl db.WatchlistUser) (field discord.EmbedField) {
	var err error

	u, ok := m[wl.UserID]
	if !ok {
		u, err = s.User(wl.UserID)
		if err == nil {
			m[wl.UserID] = u
		}
	}

	if u == nil {
		field.Name = "unknown user " + wl.UserID.String()
	} else {
		field.Name = u.Username + "#" + u.Discriminator
	}

	mod, ok := m[wl.Moderator]
	if !ok {
		mod, err = s.User(wl.Moderator)
		if err == nil {
			m[wl.Moderator] = u
		}
	}

	if mod == nil {
		field.Value = fmt.Sprintf("**Moderator:** %v", wl.Moderator.Mention())
	} else {
		field.Value = fmt.Sprintf("**Moderator:** %v#%v", mod.Username, mod.Discriminator)
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
