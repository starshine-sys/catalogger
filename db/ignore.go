package db

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/common"
)

func (db *DB) IsIgnored(id discord.MessageID) (exists bool) {
	err := db.QueryRow(context.Background(), "select exists(select * from ignored_messages where id = $1)", id).Scan(&exists)
	if err != nil {
		common.Log.Errorf("checking if message is ignored: %v", err)
	}
	return exists
}

func (db *DB) IgnoreMessage(id discord.MessageID) error {
	_, err := db.Exec(context.Background(), "insert into ignored_messages (id) values ($1) on conflict (id) do nothing", id)
	return err
}
