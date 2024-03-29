package messages

import (
	"regexp"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/db"
	"github.com/starshine-sys/pkgo/v2"
)

var ignoreApplications = [...]discord.AppID{
	discord.NullAppID, // set to bot user at runtime
}

func (bot *Bot) messageCreate(m *gateway.MessageCreateEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	channels, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		log.Errorf("getting log channels for guild %v: %v", m.GuildID, err)
		return
	}

	// if logging for these events is disabled entirely, don't save this guild's messages in the database
	if !channels.Channels.MessageUpdate.IsValid() &&
		!channels.Channels.MessageDelete.IsValid() &&
		!channels.Channels.MessageDeleteBulk.IsValid() {
		return
	}

	// create db message object
	content := m.Content
	if m.Content == "" {
		content = "None"
	}

	msg := db.Message{
		ID:        m.ID,
		UserID:    m.Author.ID,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
		Username:  m.Author.Tag(),

		Content: content,
	}

	for _, attachment := range m.Attachments {
		msg.AttachmentSize += attachment.Size
	}

	// also add extra data for bulk delete logging, if necessary
	if len(m.Embeds) > 0 || m.WebhookID.IsValid() {
		msg.Metadata = &db.Metadata{}

		if m.WebhookID.IsValid() {
			msg.Metadata.UserID = &m.Author.ID
			msg.Metadata.Username = m.Author.Username
			msg.Metadata.Avatar = m.Author.Avatar
		} else {
			msg.Metadata.Embeds = m.Embeds
		}
	}

	err = bot.DB.InsertMessage(msg)
	if err != nil {
		log.Errorf("inserting message: %v", err)
		return
	}

	// check if message was from a PluralKit webhook
	var isPK bool
	for _, id := range pkBots {
		if m.ApplicationID == discord.AppID(id) {
			isPK = true
			break
		}
	}

	// if not a proxied message, we can stop here
	if !isPK {
		return
	}

	time.Sleep(time.Second) // sleep for a second to give PK time to send the log message
	// if the message has been handled, we're done
	if bot.handledMessages.Exists(m.ID) {
		bot.handledMessages.Remove(m.ID)
		return
	}
}

var (
	pkLinkRegex   = regexp.MustCompile(`^https:\/\/discord.com\/channels\/\d+\/(\d+)\/\d+$`)
	pkFooterRegex = regexp.MustCompile(`^System ID: (\w{5,6}) \| Member ID: (\w{5,6}) \| Sender: .+ \((\d+)\) \| Message ID: (\d+) \| Original Message ID: (\d+)$`)
)

var pkBots = [...]discord.UserID{
	466378653216014359,
}

func (bot *Bot) pkMessageCreate(m *gateway.MessageCreateEvent) {
	// we only want to check PluralKit-compatible bots
	var isPK bool
	for _, id := range pkBots {
		if m.Author.ID == id {
			isPK = true
			break
		}
	}
	if !isPK {
		return
	}

	// message needs to have the content we expect
	if !pkLinkRegex.MatchString(m.Content) {
		return
	}

	// message needs to have embed with footer + match expected footer
	if len(m.Embeds) < 1 || m.Embeds[0].Footer == nil {
		return
	}

	// find matches in embed footer
	groups := pkFooterRegex.FindStringSubmatch(m.Embeds[0].Footer.Text)
	if len(groups) < 6 {
		return
	}

	var (
		sysID             = groups[1]
		memberID          = groups[2]
		userID            discord.UserID
		msgID             discord.MessageID
		originalMessageID discord.Snowflake
	)

	// all snowflakes will be valid
	{
		sf, _ := discord.ParseSnowflake(groups[3])
		userID = discord.UserID(sf)
		sf, _ = discord.ParseSnowflake(groups[4])
		msgID = discord.MessageID(sf)

		// add the original message ID to the list of proxy trigger messages
		// so that we don't log it being deleted
		originalMessageID, _ = discord.ParseSnowflake(groups[5])
		bot.proxyTriggers.Add(discord.MessageID(originalMessageID))
	}

	err := bot.DB.UpdatePKInfo(msgID, discord.MessageID(originalMessageID), pkgo.Snowflake(userID), sysID, memberID)
	if err != nil {
		log.Errorf("updating pk info for message %v: %v", msgID, err)
	}

	log.Debugf("saved pk info for message %v, original %v", msgID, originalMessageID)

	// delete the original message from the DB
	err = bot.DB.DeleteMessage(discord.MessageID(originalMessageID))
	if err != nil {
		log.Errorf("deleting original message %v from db: %v", originalMessageID, err)
	}

	// add the message ID to the list of handled messages, so that we don't call the API later
	bot.handledMessages.Add(msgID)
}
