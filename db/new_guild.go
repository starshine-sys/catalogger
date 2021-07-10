package db

import (
	"context"

	"github.com/diamondburned/arikawa/v3/gateway"
)

// CreateServerIfNotExists ...
func (db *DB) CreateServerIfNotExists(g *gateway.GuildCreateEvent) {
	var exists bool
	err := db.Pool.QueryRow(context.Background(), "select exists (select from guilds where id = $1)", g.ID).Scan(&exists)
	if err != nil {
		db.Sugar.Errorf("Error creating guild: %v", err)
		return
	}
	if !exists {
		_, err = db.Pool.Exec(context.Background(), "insert into guilds (id, channels) values ($1, $2)", g.ID, DefaultEventMap)
		if err != nil {
			db.Sugar.Errorf("Error creating guild: %v", err)
		}
	}
}
