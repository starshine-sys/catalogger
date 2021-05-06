package events

import (
	"context"
	"fmt"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) messageUpdate(m *gateway.MessageUpdateEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting channels for %v: %v", m.GuildID, err)
		return
	}

	if !ch["MESSAGE_UPDATE"].IsValid() {
		return
	}

	// if the channel is blacklisted, return
	var blacklisted bool
	if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", m.ChannelID, m.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	// try getting the message
	msg, err := bot.DB.GetProxied(m.ID)
	if err != nil {
		msg, err = bot.DB.GetMessage(m.ID)
		if err != nil {
			bot.DB.InsertMessage(db.Message{
				MsgID:     m.ID,
				UserID:    m.Author.ID,
				ChannelID: m.ChannelID,
				ServerID:  m.GuildID,
				Username:  m.Author.Username + "#" + m.Author.Discriminator,

				Content: m.Content,
			})
			return
		}
	}

	wh, err := bot.webhookCache("msg_update", m.GuildID, ch["MESSAGE_UPDATE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	mention := fmt.Sprintf("%v\n%v#%v\nID: %v", m.Author.Mention(), m.Author.Username, m.Author.Discriminator, m.Author.ID)
	author := &discord.EmbedAuthor{
		Icon: m.Author.AvatarURL(),
		Name: m.Author.Username + "#" + m.Author.Discriminator,
	}

	updated := m.Content
	if len(updated) > 1000 {
		updated = updated[:1000] + "..."
	}

	e := discord.Embed{
		Author:      author,
		Title:       fmt.Sprintf("Message by \"%v#%v\" updated\nOld content", m.Author.Username, m.Author.Discriminator),
		Description: msg.Content,
		Color:       bcr.ColourPurple,
		Fields: []discord.EmbedField{
			{
				Name:  "New content",
				Value: updated,
			},
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("%v\nID: %v", msg.ChannelID.Mention(), msg.ChannelID),
				Inline: true,
			},
			{
				Name:   "Author",
				Value:  mention,
				Inline: true,
			},
		},
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", msg.MsgID),
		},
		Timestamp: discord.NewTimestamp(msg.MsgID.Time()),
	}

	if msg.System != "" && msg.Member != "" {
		e.Title = fmt.Sprintf("Message by \"%v\" updated\nOld content", m.Author.Username)

		u, err := bot.State.User(msg.UserID)
		if err == nil {
			e.Fields[len(e.Fields)-1] = discord.EmbedField{
				Name:   "Linked Discord account",
				Value:  fmt.Sprintf("%v\n%v#%v\nID: %v", u.Mention(), u.Username, u.Discriminator, u.ID),
				Inline: true,
			}
		} else {
			e.Fields[len(e.Fields)-1] = discord.EmbedField{
				Name:   "Linked Discord account",
				Value:  fmt.Sprintf("%v\nID: %v", msg.UserID.Mention(), msg.UserID),
				Inline: true,
			}
		}

		e.Fields = append(e.Fields, []discord.EmbedField{
			{
				Name:  "​",
				Value: "**PluralKit information**",
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
		}...)
	}

	err = webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.Sugar.Errorf("Error sending message update log: %v", err)
	}

	// update the message
	if msg.System != "" {
		bot.DB.InsertProxied(db.Message{
			MsgID:     m.ID,
			UserID:    m.Author.ID,
			ChannelID: m.ChannelID,
			ServerID:  m.GuildID,

			Username: m.Author.Username,
			Member:   msg.Member,
			System:   msg.System,

			Content: m.Content,
		})
	} else {
		bot.DB.InsertMessage(db.Message{
			MsgID:     m.ID,
			UserID:    m.Author.ID,
			ChannelID: m.ChannelID,
			ServerID:  m.GuildID,
			Username:  m.Author.Username + "#" + m.Author.Discriminator,

			Content: m.Content,
		})
	}
}
