package db

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/google/uuid"
)

// IsSystemBanned returns true if the system is banned
func (db *DB) IsSystemBanned(guild discord.GuildID, id string, uuid uuid.UUID) (banned bool, err error) {
	err = db.QueryRow(context.Background(), "select $1 = any(banned_systems) or $2 = any(banned_systems) from guilds where id = $3", id, uuid.String(), guild).Scan(&banned)
	return
}

// BanSystem bans the given system
func (db *DB) BanSystem(guild discord.GuildID, uuid uuid.UUID) (err error) {
	_, err = db.Exec(context.Background(), "update guilds set banned_systems = array_append(banned_systems, $1) where id = $2", uuid.String(), guild)
	return err
}

// UnbanSystem unbans the given system
func (db *DB) UnbanSystem(guild discord.GuildID, id string, uuid uuid.UUID) (err error) {
	_, err = db.Exec(context.Background(), "update guilds set banned_systems = array_remove(array_remove(banned_systems, $1), $2) where id = $3", id, uuid.String(), guild)
	return err
}
