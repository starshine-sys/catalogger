package db

import (
	"context"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/common"
)

// CreateServerIfNotExists ...
func (db *DB) CreateServerIfNotExists(g *gateway.GuildCreateEvent) {
	var exists bool
	err := db.QueryRow(context.Background(), "select exists (select from guilds where id = $1)", g.ID).Scan(&exists)
	if err != nil {
		common.Log.Errorf("Error creating guild: %v", err)
		return
	}
	if !exists {
		_, err = db.Exec(context.Background(), "insert into guilds (id, channels) values ($1, $2)", g.ID, DefaultEventMap)
		if err != nil {
			common.Log.Errorf("Error creating guild: %v", err)
		}
	}
}
