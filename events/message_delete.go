package events

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) messageDelete(m *gateway.MessageDeleteEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting channels for %v: %v", m.GuildID, err)
		return
	}

	if !ch["MESSAGE_DELETE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("msg_delete", m.GuildID, ch["MESSAGE_DELETE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	// proxied messages are handled by a different handler
	var isProxied bool
	bot.DB.Pool.QueryRow(context.Background(), "select exists(select msg_id from pk_messages where msg_id = $1)", m.ID).Scan(&isProxied)
	if isProxied {
		return
	}

	// sleep for 5 seconds to give other handlers time to do their thing
	time.Sleep(5 * time.Second)

	// trigger messages should be ignored too
	bot.ProxiedTriggersMu.Lock()
	if _, ok := bot.ProxiedTriggers[m.ID]; ok {
		delete(bot.ProxiedTriggers, m.ID)
		bot.ProxiedTriggersMu.Unlock()
		return
	}
	bot.ProxiedTriggersMu.Unlock()

	msg, err := bot.DB.GetMessage(m.ID)
	if err != nil {
		e := discord.Embed{
			Title:       "Message deleted",
			Description: fmt.Sprintf("A message not in the database was deleted in %v (%v).", m.ChannelID.Mention(), m.ChannelID),
			Color:       bcr.ColourRed,
			Footer: &discord.EmbedFooter{
				Text: "ID: " + m.ID.String(),
			},
			Timestamp: discord.NewTimestamp(m.ID.Time()),
		}

		webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
			AvatarURL: bot.Router.Bot.AvatarURL(),
			Embeds:    []discord.Embed{e},
		})
		return
	}

	mention := msg.UserID.Mention()
	var author *discord.EmbedAuthor
	u, err := bot.State.User(msg.UserID)
	if err == nil {
		mention = fmt.Sprintf("%v\n%v#%v\nID: %v", u.Mention(), u.Username, u.Discriminator, u.ID)
		author = &discord.EmbedAuthor{
			Icon: u.AvatarURL(),
			Name: u.Username + u.Discriminator,
		}
	}

	e := discord.Embed{
		Author:      author,
		Title:       "Message deleted",
		Description: msg.Content,
		Color:       bcr.ColourRed,
		Fields: []discord.EmbedField{
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("%v\nID: %v", msg.ChannelID.Mention(), msg.ChannelID),
				Inline: true,
			},
			{
				Name:   "Sender",
				Value:  mention,
				Inline: true,
			},
		},
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", msg.MsgID),
		},
		Timestamp: discord.NewTimestamp(msg.MsgID.Time()),
	}

	_, err = webhook.New(wh.ID, wh.Token).ExecuteAndWait(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err == nil {
		bot.DB.DeleteMessage(msg.MsgID)
	}
}
