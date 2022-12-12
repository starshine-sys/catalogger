package db

import (
	"context"
	"encoding/json"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/pkgo/v2"

	"github.com/Masterminds/squirrel"
)

// Message is a single message
type Message struct {
	MsgID     discord.MessageID
	UserID    discord.UserID
	ChannelID discord.ChannelID
	ServerID  discord.GuildID

	Content  string `db:"-"`
	Username string `db:"-"`

	EncryptedContent  []byte `db:"content"`
	EncryptedUsername []byte `db:"username"`

	// These are only filled if the message was proxied by PluralKit
	Member *string
	System *string

	Metadata    *Metadata `db:"-"`
	RawMetadata *[]byte   `db:"metadata"`
}

// Metadata is optional message metadata
type Metadata struct {
	UserID   *discord.UserID `json:"user_id,omitempty"`
	Username string          `json:"username,omitempty"`
	Avatar   string          `json:"avatar,omitempty"`
	Embeds   []discord.Embed `json:"embeds,omitempty"`
}

// InsertMessage inserts a message
func (db *DB) InsertMessage(m Message) (err error) {
	if m.Content == "" {
		m.Content = "None"
	}

	m.EncryptedContent, err = encrypt([]byte(m.Content), db.aesKey)
	if err != nil {
		return errors.Wrap(err, "encrypting content")
	}

	m.EncryptedUsername, err = encrypt([]byte(m.Username), db.aesKey)
	if err != nil {
		return errors.Wrap(err, "encrypting username")
	}

	var metadata *[]byte
	if m.Metadata != nil {
		jsonb, err := json.Marshal(m.Metadata)
		if err != nil {
			return errors.Wrap(err, "marshaling metadata")
		}

		b, err := encrypt(jsonb, db.aesKey)
		if err != nil {
			return errors.Wrap(err, "encrypting metadata")
		}
		metadata = &b
	}

	_, err = db.Exec(context.Background(), `insert into messages
(msg_id, user_id, channel_id, server_id, content, username, member, system, metadata) values
($1, $2, $3, $4, $5, $6, $7, $8, $9)
on conflict (msg_id) do update
set content = $5`, m.MsgID, m.UserID, m.ChannelID, m.ServerID, m.EncryptedContent, m.EncryptedUsername, m.Member, m.System, metadata)
	return err
}

// UpdatePKInfo updates the PluralKit info for the given message, if it exists in the database.
func (db *DB) UpdatePKInfo(msgID discord.MessageID, userID pkgo.Snowflake, system, member string) (err error) {
	sql, args, err := sq.Update("messages").
		Set("user_id", userID).
		Set("system", system).
		Set("member", member).
		Where(squirrel.Eq{"msg_id": msgID}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building sql")
	}

	_, err = db.Exec(context.Background(), sql, args...)
	return
}

// UpdateUserID updates *just* the user ID for the given message, if it exists in the database.
func (db *DB) UpdateUserID(msgID discord.MessageID, userID discord.UserID) (err error) {
	sql, args, err := sq.Update("messages").
		Set("user_id", userID).
		Where(squirrel.Eq{"msg_id": msgID}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building sql")
	}

	_, err = db.Exec(context.Background(), sql, args...)
	return
}

// GetMessage gets a single message
func (db *DB) GetMessage(id discord.MessageID) (m *Message, err error) {
	m = &Message{}

	sql, args, err := sq.Select("*").
		From("messages").
		Where(squirrel.Eq{"msg_id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "building sql")
	}

	err = pgxscan.Get(context.Background(), db, m, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "getting from database")
	}

	out, err := decrypt(m.EncryptedContent, db.aesKey)
	if err != nil {
		return nil, errors.Wrap(err, "decrypting content")
	}
	m.Content = string(out)

	out, err = decrypt(m.EncryptedUsername, db.aesKey)
	if err != nil {
		return nil, errors.Wrap(err, "decrypting username")
	}
	m.Username = string(out)

	if m.RawMetadata != nil {
		b, err := decrypt(*m.RawMetadata, db.aesKey)
		if err != nil {
			log.Errorf("Error decrypting metadata for %v: %v", m.MsgID, err)
		}

		var md Metadata
		err = json.Unmarshal(b, &md)
		if err != nil {
			return m, errors.Wrap(err, "decrypting metadata")
		}
		m.Metadata = &md
	}

	return
}

// DeleteMessage deletes a message from the database
func (db *DB) DeleteMessage(id discord.MessageID) (err error) {
	sql, args, err := sq.Delete("messages").
		Where(squirrel.Eq{"msg_id": id}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building sql")
	}

	_, err = db.Exec(context.Background(), sql, args...)
	return
}
