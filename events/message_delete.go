package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

// Messages with these prefixes will get ignored
var editPrefixes = []string{"pk;edit", "pk!edit"}

func (bot *Bot) messageDelete(m *gateway.MessageDeleteEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	channel, err := bot.State(m.GuildID).Channel(m.ChannelID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageDelete,
			GuildID: m.GuildID,
		}, err)
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageDelete,
			GuildID: m.GuildID,
		}, err)
		return
	}

	if !ch[keys.MessageDelete].IsValid() {
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

	wh, err := bot.webhookCache(keys.MessageDelete, m.GuildID, ch[keys.MessageDelete])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageDelete,
			GuildID: m.GuildID,
		}, err)
		return
	}

	redirects, err := bot.DB.Redirects(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageDelete,
			GuildID: m.GuildID,
		}, err)
		return
	}

	if redirects[channelID.String()].IsValid() {
		wh, err = bot.getRedirect(m.GuildID, redirects[channelID.String()])
		if err != nil {
			bot.DB.Report(db.ErrorContext{
				Event:   keys.MessageDelete,
				GuildID: m.GuildID,
			}, err)
			return
		}
	}

	// sleep for 5 seconds to give other handlers time to do their thing
	time.Sleep(5 * time.Second)

	// trigger messages should be ignored too
	bot.ProxiedTriggersMu.Lock()
	if _, ok := bot.ProxiedTriggers[m.ID]; ok {
		bot.DB.DeleteMessage(m.ID)
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

		bot.Send(wh, keys.MessageDelete, e)
		return
	}

	// ignore any pk;edit messages
	if hasAnyPrefixLower(msg.Content, editPrefixes...) {
		bot.DB.DeleteMessage(m.ID)
		return
	}

	mention := msg.UserID.Mention()
	var author *discord.EmbedAuthor
	u, err := bot.State(m.GuildID).User(msg.UserID)
	if err == nil {
		mention = fmt.Sprintf("%v\n%v#%v\nID: %v", u.Mention(), u.Username, u.Discriminator, u.ID)
		author = &discord.EmbedAuthor{
			Icon: u.AvatarURL(),
			Name: u.Username + "#" + u.Discriminator,
		}
	}

	content := msg.Content
	if len(content) > 4000 {
		content = content[:4000] + "..."
	}

	e := discord.Embed{
		Author:      author,
		Title:       "Message deleted",
		Description: content,
		Color:       bcr.ColourRed,
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", msg.MsgID),
		},
		Timestamp: discord.NewTimestamp(msg.MsgID.Time()),
	}

	if msg.Username != "" {
		e.Title = "Message by \"" + msg.Username + "\" deleted"
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
		e.Fields[len(e.Fields)-1].Name = "Linked Discord account"

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

	bot.Queue(wh, keys.MessageDelete, e)

	err = bot.DB.DeleteMessage(msg.MsgID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageDelete,
			GuildID: m.GuildID,
		}, err)
	}
}

func hasAnyPrefixLower(s string, prefixes ...string) bool {
	s = strings.ToLower(s)

	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
