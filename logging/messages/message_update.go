package messages

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/db"
)

func (bot *Bot) messageUpdate(ev *gateway.MessageUpdateEvent) {
	if !ev.GuildID.IsValid() || !ev.Author.ID.IsValid() {
		return
	}

	// sometimes we get message update events without any content
	// so we just ignore those
	if ev.Content == "" {
		log.Debugf("got message %v with empty content, ignoring event", ev.ID)
		return
	}

	// check if the message is marked as ignored
	if ignored, err := bot.DB.IsMessageIgnored(ev.ID); ignored {
		log.Debugf("message %v is marked as ignored", ev.ID)
		return
	} else if err != nil {
		log.Errorf("checking if message %v is ignored: %v", ev.ID, err)
		return
	}

	lc, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		log.Errorf("getting channels for guild %v: %v", ev.GuildID, err)
		return
	}

	if !lc.Channels.MessageUpdate.IsValid() {
		log.Debugf("message update logs are disabled in guild %v", ev.GuildID)
		return
	}

	// check if the channel is ignored
	if common.Contains(lc.Ignores.GlobalChannels, ev.ChannelID) {
		log.Debugf("message in channel %v is ignored", ev.ChannelID)
		return
	}

	rootChannel, err := bot.Cabinet.RootChannel(context.Background(), ev.ChannelID)
	if err != nil {
		log.Errorf("getting root channel for channel %v: %v", ev.ChannelID, err)
		return
	}

	if common.Contains(lc.Ignores.GlobalChannels, rootChannel.ID) ||
		(rootChannel.ParentID.IsValid() && common.Contains(lc.Ignores.GlobalChannels, rootChannel.ParentID)) {
		log.Debugf("message in channel %v is ignored because root or category is", ev.ChannelID)
		return
	}

	// check if user is ignored globally
	if common.Contains(lc.Ignores.GlobalUsers, ev.Author.ID) {
		log.Debugf("message %v is ignored because user %v is ignored globally", ev.ID, ev.Author.ID)
		return
	}

	// check if user is ignored *in this channel*
	if common.Contains(lc.Ignores.PerChannel[ev.ChannelID.String()], ev.Author.ID) ||
		common.Contains(lc.Ignores.PerChannel[rootChannel.ID.String()], ev.Author.ID) ||
		(rootChannel.ParentID.IsValid() && common.Contains(lc.Ignores.PerChannel[rootChannel.ParentID.String()], ev.Author.ID)) {
		log.Debugf("message %v is ignored because user %v is ignored in the channel", ev.ID, ev.Author.ID)
		return
	}

	defer func() {
		content := ev.Content
		if ev.Content == "" {
			content = "None"
		}

		msg := db.Message{
			ID:        ev.ID,
			UserID:    ev.Author.ID,
			ChannelID: ev.ChannelID,
			GuildID:   ev.GuildID,
			Username:  ev.Author.Tag(),

			Content: content,
		}

		err = bot.DB.InsertMessage(msg)
		if err != nil {
			log.Errorf("updating message %v: %v", ev.ID, err)
		}
	}()

	if !bot.ShouldLog() {
		return
	}

	old, err := bot.DB.GetMessage(ev.ID)
	if err != nil {
		log.Errorf("getting old message %v: %v", ev.ID, err)
		return
	}

	if old.Content == ev.Content {
		log.Debugf("new content for message %v was identical to old content", ev.ID)
		return
	}

	embed := discord.Embed{
		Author: &discord.EmbedAuthor{
			Icon: ev.Author.AvatarURL(),
			Name: ev.Author.Tag(),
		},
		Title: "Message updated",
		Color: common.ColourPurple,
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", ev.ID),
		},
		Timestamp: discord.NewTimestamp(ev.ID.Time()),
	}

	// this entire block is needed to split the message across multiple embed fields, if it exceeds 1000 characters
	if len(old.Content) > 1000 {
		embed.Fields = append(embed.Fields, discord.EmbedField{
			Name:  "Old content",
			Value: old.Content[:1000] + "…",
		})
		if len(old.Content) > 2000 {
			if len(old.Content) > 3000 {
				val := old.Content[3000:]
				if len(val) > 500 {
					val = val[:500] + "…"
				}

				embed.Fields = append(embed.Fields, []discord.EmbedField{
					{
						Name:  "Old content (cont.)",
						Value: "…" + old.Content[1000:2000] + "…",
					},
					{
						Name:  "Old content (cont.)",
						Value: "…" + old.Content[2000:3000],
					},
					{
						Name:  "Old content (cont.)",
						Value: "…" + val,
					},
				}...)
			} else {
				embed.Fields = append(embed.Fields, []discord.EmbedField{
					{
						Name:  "Old content (cont.)",
						Value: "…" + old.Content[1000:2000] + "…",
					},
					{
						Name:  "Old content (cont.)",
						Value: "…" + old.Content[2000:],
					},
				}...)
			}
		} else {
			embed.Fields = append(embed.Fields, discord.EmbedField{
				Name:  "Old content (cont.)",
				Value: "…" + old.Content[1000:],
			})
		}
	} else {
		embed.Fields = append(embed.Fields, discord.EmbedField{
			Name:  "Old content",
			Value: old.Content,
		})
	}

	embed.Fields = append(embed.Fields, discord.EmbedField{Name: "\u200b", Value: "\u200b"})

	if len(ev.Content) > 1000 {
		embed.Fields = append(embed.Fields, discord.EmbedField{
			Name:  "New content",
			Value: ev.Content[:1000] + "…",
		})
		if len(ev.Content) > 2000 {
			val := ev.Content[1000:]
			if len(val) >= 1024 {
				val = val[:1015] + "…"
			}

			embed.Fields = append(embed.Fields, discord.EmbedField{
				Name:  "New content (cont.)",
				Value: "…" + val,
			})
		} else {
			embed.Fields = append(embed.Fields, discord.EmbedField{
				Name:  "New content (cont.)",
				Value: "…" + ev.Content[1000:],
			})
		}
	} else {
		embed.Fields = append(embed.Fields, discord.EmbedField{
			Name:  "New content",
			Value: ev.Content,
		})
	}

	// make channel mention string
	// if it's a thread, we want to add more information (thread name because they're more ephemeral than channels, plus parent channel)
	channelValue := fmt.Sprintf("%v\nID: %v", ev.ChannelID.Mention(), ev.ChannelID)
	if channel, err := bot.Cabinet.Channel(context.Background(), ev.ChannelID); err != nil {
		if common.IsThread(channel) {
			channelValue = fmt.Sprintf("%v\nID: %v\n\nThread: %v (%v)", channel.ParentID.Mention(), channel.ParentID, channel.Name, channel.Mention())
		}
	}

	embed.Fields = append(embed.Fields, []discord.EmbedField{
		{
			Name:   "Channel",
			Value:  channelValue,
			Inline: true,
		},
		{
			Name:   "Sender",
			Value:  fmt.Sprintf("%v\n%v\nID: %v", ev.Author.Mention(), ev.Author.Tag(), ev.Author.ID),
			Inline: true,
		},
	}...)

	// add PK system info
	if old.System != nil && old.Member != nil {
		embed.Title = fmt.Sprintf("Message by %v updated", ev.Author.Username)

		u, err := bot.GuildUser(ev.GuildID, old.UserID)
		if err == nil {
			embed.Author = &discord.EmbedAuthor{
				Icon: u.AvatarURLWithType(discord.PNGImage),
				Name: u.Tag(),
			}

			embed.Fields[len(embed.Fields)-1] = discord.EmbedField{
				Name:   "Linked Discord account",
				Value:  fmt.Sprintf("%v\n%v\nID: %v", u.Mention(), u.Tag(), u.ID),
				Inline: true,
			}
		} else {
			embed.Fields[len(embed.Fields)-1] = discord.EmbedField{
				Name:   "Linked Discord account",
				Value:  fmt.Sprintf("%v\nID: %v", old.UserID.Mention(), old.UserID),
				Inline: true,
			}
		}

		embed.Fields = append(embed.Fields, []discord.EmbedField{
			{
				Name:  "\u200b",
				Value: "**PluralKit information**",
			},
			{
				Name:   "System ID",
				Value:  *old.System,
				Inline: true,
			},
			{
				Name:   "Member ID",
				Value:  *old.Member,
				Inline: true,
			},
		}...)
	}

	// add link to message
	embed.Fields = append(embed.Fields, discord.EmbedField{
		Name:  "Link",
		Value: fmt.Sprintf("https://discord.com/channels/%v/%v/%v", ev.GuildID, ev.ChannelID, ev.ID),
	})

	// get the correct log channel (taking into account redirects)
	logChannel := lc.Channels.MessageUpdate
	if id, ok := lc.Redirects[ev.ChannelID.String()]; ok { // check this channel's ID
		logChannel = id
	} else if id, ok := lc.Redirects[rootChannel.ID.String()]; ok { // check root channel's ID (parent of thread)
		logChannel = id
	} else if id, ok := lc.Redirects[rootChannel.ParentID.String()]; ok && rootChannel.ParentID.IsValid() { // check root channel's parent ID (category, if in category)
		logChannel = id
	}

	if !logChannel.IsValid() {
		log.Warnf("update log for message %v in channel %v/guild %v got to end of handler, but there is no valid log channel", ev.ID, ev.ChannelID, ev.GuildID)
		return
	}

	var files []sendpart.File
	if len(ev.Content) >= 2000 || len(old.Content) >= 2000 {
		files = []sendpart.File{
			{
				Name:   "before.txt",
				Reader: strings.NewReader(old.Content),
			},
			{
				Name:   "after.txt",
				Reader: strings.NewReader(ev.Content),
			},
		}
	}

	// send log
	bot.Send(ev.GuildID, ev, SendData{
		ChannelID: logChannel,
		Embeds:    []discord.Embed{embed},
		Files:     files,
	})
}
