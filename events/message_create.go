package events

import (
	"context"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) messageCreate(m *gateway.MessageCreateEvent) {
	if !m.GuildID.IsValid() || m.WebhookID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "message_create",
			GuildID: m.GuildID,
		}, err)
		return
	}

	if !ch[keys.MessageDelete].IsValid() && !ch[keys.MessageUpdate].IsValid() && !ch[keys.MessageDeleteBulk].IsValid() {
		return
	}

	conn, err := bot.DB.Obtain()
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "message_create",
			GuildID: m.GuildID,
		}, err)
	}
	defer conn.Release()

	// if the channel is blacklisted, return
	var blacklisted bool
	if conn.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", m.ChannelID, m.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	content := m.Content
	if m.Content == "" {
		content = "None"
	}

	msg := db.Message{
		MsgID:     m.ID,
		UserID:    m.Author.ID,
		ChannelID: m.ChannelID,
		ServerID:  m.GuildID,
		Username:  m.Author.Username + "#" + m.Author.Discriminator,

		Content: content,
	}

	err = bot.DB.InsertMessage(conn, msg)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "message_create",
			GuildID: m.GuildID,
		}, err)
		return
	}
}
