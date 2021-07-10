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
	}

	if !ch["MESSAGE_DELETE"].IsValid() && !ch["MESSAGE_UPDATE"].IsValid() {
		return
	}

	// if the channels is blacklisted, return
	var blacklisted bool
	if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", m.ChannelID, m.GuildID).Scan(&blacklisted); blacklisted {
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

	err = bot.DB.InsertMessage(msg)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "message_create",
			GuildID: m.GuildID,
		}, err)
	}
}
