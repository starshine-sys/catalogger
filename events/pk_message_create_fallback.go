package events

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/pkgo"
)

var pk = pkgo.New("")

// these names are ignored if the webhook message has
// - m.Content == ""
// - len(m.Embeds) > 0
// - len(m.Attachments) == 0
var ignoreBotNames = [...]string{
	"", // changed to bot user at runtime
	"Carl-bot Logging",
	"GitHub",
}

func (bot *Bot) pkMessageCreateFallback(m *gateway.MessageCreateEvent) {
	if ignoreBotNames[0] == "" {
		ignoreBotNames[0] = bot.Bot.Username
	}

	// only check webhook messages
	if !m.WebhookID.IsValid() || !m.GuildID.IsValid() {
		return
	}

	// filter out log messages [as best as we can]
	if m.Content == "" && len(m.Embeds) > 0 && len(m.Attachments) == 0 {
		for _, name := range ignoreBotNames {
			if m.Author.Username == name {
				bot.Sugar.Debugf("Ignoring webhook message by %v", m.Author.Tag())
				return
			}
		}
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
		if err == pkgo.ErrMsgNotFound || err == pkgo.ErrNotFound {
			return
		}
		bot.Sugar.Errorf("Error getting message info from the PK API: %v", err)
		bot.DB.Report(db.ErrorContext{
			Event:   "pk_message_create_fallback",
			GuildID: m.GuildID,
		}, err)
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
