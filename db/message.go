package db

import (
	"context"
	"encoding/hex"

	"git.sr.ht/~starshine-sys/logger/crypt"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/georgysavva/scany/pgxscan"
)

// Message is a single message
type Message struct {
	MsgID     discord.MessageID
	UserID    discord.UserID
	ChannelID discord.ChannelID
	ServerID  discord.GuildID

	Content string

	// These are only filled if the message was proxied by PluralKit
	Username string
	Member   string
	System   string
}

// InsertProxied inserts a proxied message
func (db *DB) InsertProxied(m Message) (err error) {
	if m.Content == "" {
		m.Content = "None"
	}
	out, err := crypt.Encrypt([]byte(m.Content), db.AESKey)
	if err != nil {
		return err
	}
	m.Content = hex.EncodeToString(out)

	out, err = crypt.Encrypt([]byte(m.Username), db.AESKey)
	if err != nil {
		return err
	}
	m.Username = hex.EncodeToString(out)

	_, err = db.Pool.Exec(context.Background(), `insert into pk_messages
(msg_id, user_id, channel_id, server_id, username, member, system, content) values
($1, $2, $3, $4, $5, $6, $7, $8)`, m.MsgID, m.UserID, m.ChannelID, m.ServerID, m.Username, m.Member, m.System, m.Content)
	return err
}

// GetProxied gets a single proxied message
func (db *DB) GetProxied(id discord.MessageID) (m *Message, err error) {
	m = &Message{}

	err = pgxscan.Get(context.Background(), db.Pool, m, "select * from pk_messages where msg_id = $1", id)

	b, err := hex.DecodeString(m.Content)
	if err != nil {
		return nil, err
	}

	out, err := crypt.Decrypt(b, db.AESKey)
	if err != nil {
		return nil, err
	}

	m.Content = string(out)

	b, err = hex.DecodeString(m.Username)
	if err != nil {
		return nil, err
	}

	out, err = crypt.Decrypt(b, db.AESKey)
	if err != nil {
		return nil, err
	}

	m.Username = string(out)
	return
}

// DeleteProxied deletes a proxied message from the database
func (db *DB) DeleteProxied(id discord.MessageID) (err error) {
	_, err = db.Pool.Exec(context.Background(), "delete from pk_messages where msg_id = $1", id)
	return
}
