package importdb

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/db"
)

type OldMessage struct {
	MsgID     discord.MessageID
	UserID    discord.UserID
	ChannelID discord.ChannelID
	ServerID  discord.GuildID

	Content  string
	Username string

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

// GetOldMessages gets 1000 messages from the old database
func GetOldMessages(conn *pgx.Conn, key [32]byte, prevMaxID discord.MessageID) (ms []OldMessage, err error) {
	sql, args, err := sq.Select("*").
		From("messages").
		Where("msg_id > ?", prevMaxID).
		OrderBy("msg_id ASC").
		Limit(1000).
		ToSql()
	if err != nil {
		return ms, errors.Wrap(err, "building sql")
	}

	err = pgxscan.Select(context.Background(), conn, &ms, sql, args...)
	if err != nil {
		return ms, errors.Wrap(err, "getting message")
	}

	for i, m := range ms {
		b, err := hex.DecodeString(m.Content)
		if err != nil {
			return ms, errors.Wrap(err, "")
		}

		out, err := db.Decrypt(b, key)
		if err != nil {
			return ms, err
		}

		ms[i].Content = string(out)

		b, err = hex.DecodeString(m.Username)
		if err != nil {
			return ms, err
		}

		out, err = db.Decrypt(b, key)
		if err != nil {
			return ms, err
		}

		ms[i].Username = string(out)

		if m.RawMetadata != nil {
			b, err := db.Decrypt(*m.RawMetadata, key)
			if err != nil {
				return ms, errors.Wrap(err, "decrypting metadata")
			}

			var md Metadata
			err = json.Unmarshal(b, &md)
			if err != nil {
				return ms, errors.Wrap(err, "decrypting metadata")
			}
			ms[i].Metadata = &md
		}
	}

	return
}

type NewMessage = db.Message
type NewMetadata = db.Metadata

func InsertNewMessage(conn *pgx.Conn, key [32]byte, oldMessage OldMessage) (err error) {
	m := NewMessage{
		ID:        oldMessage.MsgID,
		UserID:    oldMessage.UserID,
		ChannelID: oldMessage.ChannelID,
		GuildID:   oldMessage.ServerID,

		Content:  oldMessage.Content,
		Username: oldMessage.Username,
		Member:   oldMessage.Member,
		System:   oldMessage.System,

		Metadata: (*NewMetadata)(oldMessage.Metadata),

		AttachmentSize: 0,
	}

	if m.Content == "" {
		m.Content = "None"
	}

	m.EncryptedContent, err = db.Encrypt([]byte(m.Content), key)
	if err != nil {
		return errors.Wrap(err, "encrypting content")
	}

	m.EncryptedUsername, err = db.Encrypt([]byte(m.Username), key)
	if err != nil {
		return errors.Wrap(err, "encrypting username")
	}

	var metadata *[]byte
	if m.Metadata != nil {
		jsonb, err := json.Marshal(m.Metadata)
		if err != nil {
			return errors.Wrap(err, "marshaling metadata")
		}

		b, err := db.Encrypt(jsonb, key)
		if err != nil {
			return errors.Wrap(err, "encrypting metadata")
		}
		metadata = &b
	}

	_, err = conn.Exec(context.Background(), `insert into messages
(id, user_id, channel_id, guild_id, content, username, member, system, metadata) values
($1, $2, $3, $4, $5, $6, $7, $8, $9)
on conflict (id) do update
set content = $5, metadata = $9`, m.ID, m.UserID, m.ChannelID, m.GuildID, m.EncryptedContent, m.EncryptedUsername, m.Member, m.System, metadata)
	return err
}

func doMigrateMessages(oldDB *pgx.Conn, newDB *pgx.Conn, oldKey [32]byte, newKey [32]byte) (err error) {
	log.Info("copying messages")

	var (
		msgStartTime  = time.Now()
		msgsCopied    = 0
		lastMessageID = discord.MessageID(0)
	)

	for {
		sectionStart := time.Now()

		msgs, err := GetOldMessages(oldDB, oldKey, lastMessageID)
		if err != nil {
			return errors.Wrapf(err, "getting 1000 messages after %v", lastMessageID)
		}

		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			err = InsertNewMessage(newDB, newKey, msg)
			if err != nil {
				return errors.Wrapf(err, "getting message %v", msg.MsgID)
			}
		}

		log.Infof("copied %v messages (from %v to %v) in %v", len(msgs), msgs[0].MsgID, msgs[len(msgs)-1].MsgID, time.Since(sectionStart))

		lastMessageID = msgs[len(msgs)-1].MsgID
		msgsCopied += len(msgs)
	}

	log.Infof("finished copying messages in %v", time.Since(msgStartTime))
	return nil
}
