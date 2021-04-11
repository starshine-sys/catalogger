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

func (bot *Bot) pkMessageDelete(m *gateway.MessageDeleteEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["MESSAGE_DELETE"].IsValid() {
		return
	}

	// if the channels is blacklisted, return
	var blacklisted bool
	if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2", m.ChannelID, m.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	// try getting the message
	msg, err := bot.DB.GetProxied(m.ID)
	if err != nil {
		return
	}

	wh, err := bot.webhookCache("msg_delete", m.GuildID, ch["MESSAGE_DELETE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
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
		Title:       fmt.Sprintf("Message by \"%v\" deleted", msg.Username),
		Description: msg.Content,
		Color:       bcr.ColourRed,
		Fields: []discord.EmbedField{
			{
				Name:  "​",
				Value: "​",
			},
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("%v\nID: %v", msg.ChannelID.Mention(), msg.ChannelID),
				Inline: true,
			},
			{
				Name:   "Linked Discord account",
				Value:  mention,
				Inline: true,
			},
			{
				Name:  "​",
				Value: "​",
			},
			{
				Name:   "System ID",
				Value:  msg.System,
				Inline: true,
			},
			{
				Name:   "Member ID",
				Value:  msg.Member,
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
		// give other message delete handler time to check the database
		time.Sleep(1 * time.Second)
		bot.DB.DeleteProxied(msg.MsgID)
	}
}
