package db

import (
	"context"
	"encoding/hex"

	"github.com/Masterminds/squirrel"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/starshine-sys/catalogger/crypt"
)

type Invite struct {
	Code, Name string
}

type Invites []Invite

func (i Invites) Name(code string) string {
	for _, i := range i {
		if i.Code == code {
			return i.Name
		}
	}
	return "Unnamed"
}

// GetInvites gets this guild's named invites
func (db *DB) GetInvites(guildID discord.GuildID) (Invites, error) {
	sql, args, err := sq.Select("code", "name").From("invites").Where(squirrel.Eq{"guild_id": guildID}).ToSql()
	if err != nil {
		return nil, err
	}

	var list []Invite

	err = pgxscan.Select(context.Background(), db, &list, sql, args...)
	if err != nil {
		return nil, err
	}

	for i := range list {
		b, err := hex.DecodeString(list[i].Name)
		if err != nil {
			return nil, err
		}

		out, err := crypt.Decrypt(b, db.AESKey)
		if err != nil {
			return nil, err
		}

		list[i].Name = string(out)
	}
	return list, err
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

	_, err = db.Exec(context.Background(), sql, args...)
	return err
}

// GetInviteName gets an invite by name.
// If the invite is not found, returns "Unnamed".
func (db *DB) GetInviteName(code string) (name string, err error) {
	sql, args, err := sq.Select("name").From("invites").Where(squirrel.Eq{"code": code}).ToSql()
	if err != nil {
		return "Unnamed", err
	}

	err = db.QueryRow(context.Background(), sql, args...).Scan(&name)
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
