package db

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/georgysavva/scany/pgxscan"
)

// WatchlistUser is a single user on the watchlist.
type WatchlistUser struct {
	GuildID discord.GuildID
	UserID  discord.UserID

	Moderator discord.UserID
	Added     time.Time

	Reason string
}

// GuildWatchlist gets the watchlist for the given guild ID.
func (db *DB) GuildWatchlist(id discord.GuildID) (l []WatchlistUser, err error) {
	err = pgxscan.Select(context.Background(), db.Pool, &l, "select * from watchlist where guild_id = $1 order by user_id", id)
	return
}

// WatchlistAdd adds a user to the watchlist, or updates the reason.
func (db *DB) WatchlistAdd(guildID discord.GuildID, userID, modID discord.UserID, reason string) (w WatchlistUser, err error) {
	err = pgxscan.Get(context.Background(), db.Pool, &w, `insert into watchlist (guild_id, user_id, moderator, added, reason) values
	($1, $2, $3, $4, $5)
	on conflict (guild_id, user_id) do update
	set moderator = $3, added = $4, reason = $5 returning *`, guildID, userID, modID, time.Now().UTC(), reason)
	return
}

// WatchlistRemove removes a user from the watchlist.
func (db *DB) WatchlistRemove(guildID discord.GuildID, userID discord.UserID) (err error) {
	_, err = db.Pool.Exec(context.Background(), "delete from watchlist where guild_id = $1 and user_id = $2", guildID, userID)
	return
}

// UserWatchlist returns the watchlist entry for a user, if any.
func (db *DB) UserWatchlist(guildID discord.GuildID, userID discord.UserID) (w *WatchlistUser, err error) {
	w = &WatchlistUser{}
	err = pgxscan.Get(context.Background(), db.Pool, w, "select * from watchlist where guild_id = $1 and user_id = $2", guildID, userID)
	if err != nil {
		return nil, err
	}
	return w, nil
}
