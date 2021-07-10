package db

import (
	"context"
	"encoding/hex"

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

	err = pgxscan.Select(context.Background(), db.Pool, &list, "select code, name from invites where guild_id = $1", guildID)
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

	_, err = db.Pool.Exec(context.Background(), `insert into invites (guild_id, code, name)
values ($1, $2, $3) on conflict (code)
do update set name = $3`, guildID, code, name)
	return err
}

// GetInviteName gets an invite by name.
// If the invite is not found, returns "Unnamed".
func (db *DB) GetInviteName(code string) (name string, err error) {
	err = db.Pool.QueryRow(context.Background(), "select name from invites where code = $1", code).Scan(&name)
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
