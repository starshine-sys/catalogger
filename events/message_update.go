package events

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) messageUpdate(m *gateway.MessageUpdateEvent) {
	if !m.GuildID.IsValid() || !m.Author.ID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "message_update",
			GuildID: m.GuildID,
		}, err)
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
		bot.DB.Report(db.ErrorContext{
			Event:   "message_update",
			GuildID: m.GuildID,
		}, err)
	}

	redirects, err := bot.DB.Redirects(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "message_update",
			GuildID: m.GuildID,
		}, err)
	}

	if redirects[m.ChannelID.String()].IsValid() {
		wh, err = bot.getRedirect(m.GuildID, redirects[m.ChannelID.String()])
		if err != nil {
			bot.DB.Report(db.ErrorContext{
				Event:   "message_update",
				GuildID: m.GuildID,
			}, err)
		}
	}

	mention := fmt.Sprintf("%v\n%v#%v\nID: %v", m.Author.Mention(), m.Author.Username, m.Author.Discriminator, m.Author.ID)
	author := &discord.EmbedAuthor{
		Icon: m.Author.AvatarURL(),
		Name: m.Author.Username + "#" + m.Author.Discriminator,
	}

	e := discord.Embed{
		Author: author,
		Title:  fmt.Sprintf("Message by \"%v#%v\" updated", m.Author.Username, m.Author.Discriminator),
		Color:  bcr.ColourPurple,
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", msg.MsgID),
		},
		Timestamp: discord.NewTimestamp(msg.MsgID.Time()),
	}

	updated := m.Content
	if updated == "" {
		updated = "None"
	}

	// sometimes we get update events that don't actually change the content
	// including stuff like the message getting pinned
	// so we just ignore those updates
	if updated == msg.Content {
		return
	}

	if len(msg.Content) > 1000 {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Old content",
			Value: msg.Content[:1000] + "...",
		})
		if len(msg.Content) > 2000 {
			if len(msg.Content) > 3000 {
				val := msg.Content[3000:]
				if len(val) > 500 {
					val = val[:500] + "..."
				}

				e.Fields = append(e.Fields, []discord.EmbedField{
					{
						Name:  "Old content (cont.)",
						Value: "..." + msg.Content[1000:2000] + "...",
					},
					{
						Name:  "Old content (cont.)",
						Value: "..." + msg.Content[2000:3000],
					},
					{
						Name:  "Old content (cont.)",
						Value: "..." + val,
					},
				}...)
			} else {
				e.Fields = append(e.Fields, []discord.EmbedField{
					{
						Name:  "Old content (cont.)",
						Value: "..." + msg.Content[1000:2000] + "...",
					},
					{
						Name:  "Old content (cont.)",
						Value: "..." + msg.Content[2000:],
					},
				}...)
			}
		} else {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Old content (cont.)",
				Value: "..." + msg.Content[1000:],
			})
		}
	} else {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Old content",
			Value: msg.Content,
		})
	}

	e.Fields = append(e.Fields, discord.EmbedField{Name: "​", Value: "​"})

	if len(updated) > 1000 {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "New content",
			Value: updated[:1000] + "...",
		})
		if len(updated) > 2000 {
			val := updated[1000:]
			if len(val) > 1024 {
				val = val[:1015] + "..."
			}

			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "New content (cont.)",
				Value: "..." + val,
			})
		} else {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Old content (cont.)",
				Value: "..." + updated[1000:],
			})
		}
	} else {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "New content",
			Value: updated,
		})
	}

	e.Fields = append(e.Fields, []discord.EmbedField{
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
	}...)

	if msg.System != "" && msg.Member != "" {
		e.Title = fmt.Sprintf("Message by \"%v\" updated", m.Author.Username)

		u, err := bot.State(m.GuildID).User(msg.UserID)
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

	e.Fields = append(e.Fields, discord.EmbedField{
		Name:  "Link",
		Value: fmt.Sprintf("https://discord.com/channels/%v/%v/%v", m.GuildID, m.ChannelID, m.ID),
	})

	client := webhook.New(wh.ID, wh.Token)
	bot.Queue(wh, "message_update", client, e)

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
