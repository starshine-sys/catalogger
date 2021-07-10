package events

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/pkgo"
)

var pk = pkgo.New("")

func (bot *Bot) pkMessageCreateFallback(m *gateway.MessageCreateEvent) {
	// only check webhook messages
	if !m.WebhookID.IsValid() || !m.GuildID.IsValid() {
		return
	}

	// wait 2 seconds
	time.Sleep(2 * time.Second)

	// if the channel is blacklisted, return
	var blacklisted bool
	if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", m.ChannelID, m.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	// check if the message exists in the database; if so, return
	_, err := bot.DB.GetProxied(m.ID)
	if err == nil {
		return
	}

	pkm, err := pk.Message(pkgo.Snowflake(m.ID))
	if err != nil {
		// Message is either not proxied or we got an error from the PK API. Either way, return
		return
	}

	bot.ProxiedTriggersMu.Lock()
	bot.ProxiedTriggers[discord.MessageID(pkm.Original)] = struct{}{}
	bot.ProxiedTriggersMu.Unlock()

	msg := db.Message{
		MsgID:     m.ID,
		UserID:    discord.UserID(pkm.Sender),
		ChannelID: m.ChannelID,
		ServerID:  m.GuildID,

		Username: m.Author.Username,
		Member:   pkm.Member.ID,
		System:   pkm.System.ID,

		Content: m.Content,
	}

	// insert the message, ignore errors as those shouldn't impact anything
	bot.DB.InsertProxied(msg)
}
