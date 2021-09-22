package events

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jackc/pgx/v4"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) messageUpdate(m *gateway.MessageUpdateEvent) {
	if !m.GuildID.IsValid() || !m.Author.ID.IsValid() {
		return
	}

	// sometimes we get message update events without any content
	// so just ignore those
	if m.Content == "" {
		return
	}

	channel, err := bot.State(m.GuildID).Channel(m.ChannelID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageUpdate,
			GuildID: m.GuildID,
		}, err)
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageUpdate,
			GuildID: m.GuildID,
		}, err)
		return
	}

	if !ch[keys.MessageUpdate].IsValid() {
		return
	}

	// if the channel is blacklisted, return
	channelID := m.ChannelID
	if channel.Type == discord.GuildNewsThread || channel.Type == discord.GuildPrivateThread || channel.Type == discord.GuildPublicThread {
		channelID = channel.ParentID
	}
	var blacklisted bool
	if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", channelID, m.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	// try getting the message
	msg, err := bot.DB.GetMessage(m.ID)
	if err != nil {
		if errors.Cause(err) != pgx.ErrNoRows {
			bot.Sugar.Errorf("Error fetching message ID %v from database: %v", m.ID, err)
		}

		// insert message and return
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

	wh, err := bot.webhookCache(keys.MessageUpdate, m.GuildID, ch[keys.MessageUpdate])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageUpdate,
			GuildID: m.GuildID,
		}, err)
		return
	}

	redirects, err := bot.DB.Redirects(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageUpdate,
			GuildID: m.GuildID,
		}, err)
		return
	}

	if redirects[channelID.String()].IsValid() {
		wh, err = bot.getRedirect(m.GuildID, redirects[channelID.String()])
		if err != nil {
			bot.DB.Report(db.ErrorContext{
				Event:   keys.MessageUpdate,
				GuildID: m.GuildID,
			}, err)
			return
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

	value := fmt.Sprintf("%v\nID: %v", msg.ChannelID.Mention(), msg.ChannelID)
	if channel.Type == discord.GuildNewsThread || channel.Type == discord.GuildPrivateThread || channel.Type == discord.GuildPublicThread {
		value = fmt.Sprintf("%v\nID: %v\n\nThread: %v (%v)", channel.ParentID.Mention(), channel.ParentID, channel.Name, channel.Mention())
	}

	e.Fields = append(e.Fields, []discord.EmbedField{
		{
			Name:   "Channel",
			Value:  value,
			Inline: true,
		},
		{
			Name:   "Sender",
			Value:  mention,
			Inline: true,
		},
	}...)

	if msg.System != nil && msg.Member != nil {
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
				Value:  *msg.System,
				Inline: true,
			},
			{
				Name:   "Member ID",
				Value:  *msg.Member,
				Inline: true,
			},
		}...)
	}

	e.Fields = append(e.Fields, discord.EmbedField{
		Name:  "Link",
		Value: fmt.Sprintf("https://discord.com/channels/%v/%v/%v", m.GuildID, m.ChannelID, m.ID),
	})

	bot.Queue(wh, keys.MessageUpdate, e)

	// update the message
	username := m.Author.Username
	if msg.System == nil {
		username += "#" + m.Author.Discriminator
	}

	bot.DB.InsertMessage(db.Message{
		MsgID:     m.ID,
		UserID:    m.Author.ID,
		ChannelID: m.ChannelID,
		ServerID:  m.GuildID,
		Username:  username,
		Member:    msg.Member,
		System:    msg.System,

		Content: m.Content,
	})
}
