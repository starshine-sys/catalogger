package events

import (
	"time"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/pkgo"
)

var pk = pkgo.NewSession(nil)

func (bot *Bot) pkMessageCreateFallback(m *gateway.MessageCreateEvent) {
	// only check webhook messages
	if !m.WebhookID.IsValid() || !m.GuildID.IsValid() {
		return
	}

	// wait 2 seconds
	time.Sleep(2 * time.Second)

	// check if the message exists in the database; if so, return
	_, err := bot.DB.GetProxied(m.ID)
	if err == nil {
		return
	}

	pkm, err := pk.GetMessage(m.ID.String())
	if err != nil {
		// Message is either not proxied or we got an error from the PK API. Either way, return
		return
	}

	u, _ := discord.ParseSnowflake(pkm.Sender)

	orig, _ := discord.ParseSnowflake(pkm.Original)
	bot.ProxiedTriggersMu.Lock()
	bot.ProxiedTriggers[discord.MessageID(orig)] = struct{}{}
	bot.ProxiedTriggersMu.Unlock()

	msg := db.Message{
		MsgID:     m.ID,
		UserID:    discord.UserID(u),
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
