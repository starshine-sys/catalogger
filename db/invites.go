package db

import (
	"context"
	"encoding/hex"

	"github.com/Masterminds/squirrel"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/starshine-sys/catalogger/crypt"
)

// GetInvites gets this guild's named invites
func (db *DB) GetInvites(guildID discord.GuildID, invites map[string]string) (names map[string]string, err error) {
	var list []struct {
		Code string
		Name string
	}

	sql, args, err := sq.Select("code", "name").From("invites").Where(squirrel.Eq{"guild_id": 1}).ToSql()
	if err != nil {
		return nil, err
	}

	err = pgxscan.Select(context.Background(), db.Pool, &list, sql, args...)
	if err != nil {
		return
	}

	for _, i := range list {
		b, err := hex.DecodeString(i.Name)
		if err != nil {
			return nil, err
		}

		out, err := crypt.Decrypt(b, db.AESKey)
		if err != nil {
			return nil, err
		}

		i.Name = string(out)

		invites[i.Code] = i.Name
	}
	return invites, err
}

// NameInvite names an invite
func (db *DB) NameInvite(guildID discord.GuildID, code, name string) (err error) {
	out, err := crypt.Encrypt([]byte(name), db.AESKey)
	if err != nil {
		return err
	}
	name = hex.EncodeToString(out)

	sql, args, err := sq.Insert("invites").Columns("guild_id", "code", "name").Values(guildID, code, name).Suffix("ON CONFLICT (code) DO UPDATE SET name = ?", name).ToSql()
	if err != nil {
		return err
	}

	_, err = db.Pool.Exec(context.Background(), sql, args...)
	return err
}

// GetInviteName gets an invite by name.
// If the invite is not found, returns "Unnamed".
func (db *DB) GetInviteName(code string) (name string, err error) {
	sql, args, err := sq.Select("name").From("invites").Where(squirrel.Eq{"code": code}).ToSql()
	if err != nil {
		return "Unnamed", err
	}

	err = db.Pool.QueryRow(context.Background(), sql, args...).Scan(&name)
	if err != nil {
		return "Unnamed", nil
	}

	b, err := hex.DecodeString(name)
	if err != nil {
		return "Unnamed", err
	}

	out, err := crypt.Decrypt(b, db.AESKey)
	if err != nil {
		return "Unnamed", err
	}

	return string(out), err
}
