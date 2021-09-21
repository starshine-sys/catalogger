package db

import (
	"context"
	"encoding/hex"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/starshine-sys/catalogger/crypt"

	"github.com/Masterminds/squirrel"
)

var sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// Message is a single message
type Message struct {
	MsgID     discord.MessageID
	UserID    discord.UserID
	ChannelID discord.ChannelID
	ServerID  discord.GuildID

	Content  string
	Username string

	// These are only filled if the message was proxied by PluralKit
	Member *string
	System *string
}

// InsertMessage inserts a message
func (db *DB) InsertMessage(m Message) (err error) {
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

	_, err = db.Pool.Exec(context.Background(), `insert into messages
(msg_id, user_id, channel_id, server_id, content, username, member, system) values
($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (msg_id) do update
set content = $5`, m.MsgID, m.UserID, m.ChannelID, m.ServerID, m.Content, m.Username, m.Member, m.System)
	return err
}

// GetMessage gets a single message
func (db *DB) GetMessage(id discord.MessageID) (m *Message, err error) {
	m = &Message{}

	sql, args, err := sq.Select("*").From("messages").Where(squirrel.Eq{"msg_id": id}).ToSql()
	if err != nil {
		return nil, err
	}
	err = pgxscan.Get(context.Background(), db.Pool, m, sql, args...)

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

// DeleteMessage deletes a message from the database
func (db *DB) DeleteMessage(id discord.MessageID) (err error) {
	_, err = db.Pool.Exec(context.Background(), "delete from messages where msg_id = $1", id)
	return
}
