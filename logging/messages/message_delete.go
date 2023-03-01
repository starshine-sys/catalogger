package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/pkgo/v2"
)

func (bot *Bot) messageDelete(ev *gateway.MessageDeleteEvent) {
	// PluralKit triggers should always be deleted within 30 seconds of the message being sent
	// if it's possibly a trigger message, we wait 2 seconds to give the message create handler time to catch up
	if ev.ID.Time().After(time.Now().Add(-30 * time.Second)) {
		time.Sleep(time.Second)
	}

	if !ev.GuildID.IsValid() {
		log.Debugf("message with ID %v is in DMs, ignoring", ev.ID)
		return
	}

	if bot.proxyTriggers.Exists(ev.ID) {
		log.Debugf("message with ID %v is a proxy trigger message, ignoring", ev.ID)
		bot.proxyTriggers.Remove(ev.ID)
		return
	}

	// check if message exists in db and if it's already got pluralkit info
	if !bot.DB.HasPKInfo(ev.ID) {
		log.Debugf("fetching PK info for message %v", ev.ID)

		pkm, err := bot.PK.Message(pkgo.Snowflake(ev.ID))
		if err == nil {
			if pkm.ID != pkgo.Snowflake(ev.ID) {
				log.Debugf("message with ID %v is a proxy trigger message, saving PK info and ignoring delete", ev.ID)

				if pkm.System == nil || pkm.Member == nil {
					log.Debugf("PluralKit info for message %v has nil system or member, ignoring", pkm.ID)
					return
				}

				// update info in db
				err = bot.DB.UpdatePKInfo(discord.MessageID(pkm.ID), ev.ID, pkm.Sender, pkm.System.ID, pkm.Member.ID)
				if err != nil {
					log.Errorf("updating PluralKit API info for %v: %v", pkm.ID, err)
				}

				// delete original message
				err = bot.DB.DeleteMessage(ev.ID)
				if err != nil {
					log.Errorf("deleting original proxy trigger message %v: %v", ev.ID, err)
				}
				return
			}
		} else {
			pkerr, ok := err.(*pkgo.PKAPIError)
			if ok && pkerr.Code == pkgo.MessageNotFound {
				log.Debugf("message %v is not a proxy or proxy trigger message", ev.ID)
			} else {
				log.Errorf("getting PluralKit API info for message %v: %v", ev.ID, err)
			}
		}
	}

	lc, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		log.Errorf("getting channels for guild %v: %v", ev.GuildID, err)
		return
	}

	if !lc.Channels.MessageDelete.IsValid() {
		log.Debugf("message delete logs are disabled in guild %v", ev.GuildID)
		return
	}

	defer func() {
		err = bot.DB.DeleteMessage(ev.ID)
		if err != nil {
			log.Errorf("deleting message %v from db: %v", ev.ID, err)
		}
	}()

	if !bot.ShouldLog() {
		return
	}

	m, err := bot.DB.GetMessage(ev.ID)
	if err != nil {
		log.Errorf("getting message object for %v from db: %v", ev.ID, err)
		return
	}

	// check if the channel is ignored
	if common.Contains(lc.Ignores.GlobalChannels, m.ChannelID) {
		log.Debugf("message in channel %v is ignored", m.ChannelID)
		return
	}

	rootChannel, err := bot.Cabinet.RootChannel(context.Background(), m.ChannelID)
	if err != nil {
		log.Errorf("getting root channel for channel %v: %v", m.ChannelID, err)
		return
	}

	if common.Contains(lc.Ignores.GlobalChannels, rootChannel.ID) ||
		(rootChannel.ParentID.IsValid() && common.Contains(lc.Ignores.GlobalChannels, rootChannel.ParentID)) {
		log.Debugf("message in channel %v is ignored because root or category is", m.ChannelID)
		return
	}

	// check if user is ignored globally
	if common.Contains(lc.Ignores.GlobalUsers, m.UserID) {
		log.Debugf("message %v is ignored because user %v is ignored globally", m.ID, m.UserID)
		return
	}

	// check if user is ignored *in this channel*
	if common.Contains(lc.Ignores.PerChannel[m.ChannelID.String()], m.UserID) ||
		common.Contains(lc.Ignores.PerChannel[rootChannel.ID.String()], m.UserID) ||
		(rootChannel.ParentID.IsValid() && common.Contains(lc.Ignores.PerChannel[rootChannel.ParentID.String()], m.UserID)) {
		log.Debugf("message %v is ignored because user %v is ignored in the channel", m.ID, m.UserID)
		return
	}

	// everything checks out, start building embed!
	embed := discord.Embed{
		Title:       fmt.Sprintf("Message by %v deleted", m.Username),
		Description: m.Content,
		Color:       common.ColourRed,
		Footer: &discord.EmbedFooter{
			Text: "ID: " + m.ID.String(),
		},
		Timestamp: discord.Timestamp(m.ID.Time()),
	}

	// fetch user
	u, err := bot.GuildUser(m.GuildID, m.UserID)
	if err != nil {
		u = &discord.User{
			Username:      "unknown",
			Discriminator: "0000",
			ID:            m.UserID,
		}
	}
	embed.Author = &discord.EmbedAuthor{
		Name: u.Tag(),
		Icon: u.AvatarURL(),
	}

	// make channel field
	ch, err := bot.Cabinet.Channel(context.Background(), m.ChannelID)
	if err != nil {
		ch = discord.Channel{
			ID:   m.ChannelID,
			Name: "unknown",
		}
	}

	channelField := discord.EmbedField{
		Name:   "Channel",
		Value:  fmt.Sprintf("%v\nID: %v", ch.Mention(), ch.ID),
		Inline: true,
	}

	if common.IsThread(ch) {
		channelField.Value = fmt.Sprintf("%v\nID: %v\n\n**Thread**\n%v\nID: %v", rootChannel.Mention(), rootChannel.ID, ch.Name, ch.ID)
	}
	embed.Fields = append(embed.Fields, channelField)

	// make user field
	userField := discord.EmbedField{
		Name:   "Author",
		Value:  fmt.Sprintf("%v\n%v\nID: %v", u.Mention(), u.Tag(), u.ID),
		Inline: true,
	}
	if m.System != nil && m.Member != nil {
		userField.Name = "Linked Discord account"
	}
	embed.Fields = append(embed.Fields, userField)

	// add PluralKit information
	// these fields will always *both* be null or *both* be non-null
	if m.System != nil && m.Member != nil {
		embed.Fields = append(embed.Fields, []discord.EmbedField{
			{Name: "\u200b", Value: "**PluralKit information**"},
			{Name: "System ID", Value: *m.System, Inline: true},
			{Name: "Member ID", Value: *m.Member, Inline: true},
		}...)
	}

	// get the correct log channel (taking into account redirects)
	logChannel := lc.Channels.MessageDelete
	if id, ok := lc.Redirects[m.ChannelID.String()]; ok { // check this channel's ID
		logChannel = id
	} else if id, ok := lc.Redirects[rootChannel.ID.String()]; ok { // check root channel's ID (parent of thread)
		logChannel = id
	} else if id, ok := lc.Redirects[rootChannel.ParentID.String()]; ok && rootChannel.ParentID.IsValid() { // check root channel's parent ID (category, if in category)
		logChannel = id
	}

	if !logChannel.IsValid() {
		log.Warnf("delete log for message %v in channel %v/guild %v got to end of handler, but there is no valid log channel", m.ID, m.ChannelID, m.GuildID)
		return
	}

	// send log message!
	bot.Send(m.GuildID, ev, SendData{
		ChannelID: logChannel,
		Embeds:    []discord.Embed{embed},
	})
}
