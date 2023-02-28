package db

import (
	"context"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
)

func (db *DB) CreateGuild(id discord.GuildID) (alreadyExists bool, err error) {
	sql, args, err := sq.Insert("guilds").
		Columns("id", "channels").
		Values(id, LogChannels{}).
		Suffix("ON CONFLICT DO NOTHING").
		ToSql()
	if err != nil {
		return false, errors.Wrap(err, "building sql")
	}

	ct, err := db.Exec(context.Background(), sql, args...)
	if err != nil {
		return false, errors.Wrap(err, "executing query")
	}

	return ct.RowsAffected() == 0, nil
}
