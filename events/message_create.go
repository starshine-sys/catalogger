package events

import (
	"regexp"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events/handler"
	"github.com/starshine-sys/pkgo/v2"
)

var pk = pkgo.New("")

// these names are ignored if the webhook message has
// - m.Content == ""
// - len(m.Embeds) > 0
// - len(m.Attachments) == 0
var ignoreBotNames = [...]string{
	"", // changed to bot user at runtime
	"Carl-bot Logging",
	"GitHub",
}

func (bot *Bot) messageCreate(m *gateway.MessageCreateEvent) (*handler.Response, error) {
	if !m.GuildID.IsValid() {
		return nil, nil
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		return nil, err
	}

	if !ch[keys.MessageDelete].IsValid() && !ch[keys.MessageUpdate].IsValid() && !ch[keys.MessageDeleteBulk].IsValid() {
		return nil, nil
	}

	// if the channel is ignored, return
	channel, err := bot.RootChannel(m.GuildID, m.ChannelID)
	if err != nil {
		return nil, err
	}

	if bot.DB.IsBlacklisted(m.GuildID, channel.ID) {
		err = bot.DB.IgnoreMessage(m.ID)
		return nil, err
	}

	if bot.isUserIgnored(m.GuildID, m.Author.ID) {
		common.Log.Debugf("user %v is ignored in guild %v", m.Author.ID, m.GuildID)

		err = bot.DB.IgnoreMessage(m.ID)
		return nil, err
	}

	for _, id := range pkBotsToCheck {
		if m.Author.ID == id {
			_, err := bot.pkMessageCreate(m)
			if err != nil {
				common.Log.Errorf("Error parsing possible PluralKit proxy log: %v", err)
			}
		}
	}

	content := m.Content
	if m.Content == "" {
		content = "None"
	}

	msg := db.Message{
		MsgID:     m.ID,
		UserID:    m.Author.ID,
		ChannelID: m.ChannelID,
		ServerID:  m.GuildID,
		Username:  m.Author.Username + "#" + m.Author.Discriminator,

		Content: content,
	}

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
		return nil, err
	}

	if !m.WebhookID.IsValid() {
		return nil, nil
	}

	// set bot name
	if ignoreBotNames[0] == "" {
		ignoreBotNames[0] = bot.Router.Bot.Username
	}

	// filter out log messages [as best as we can]
	// - message with content or attachments is assumed to be a proxied message
	// - message must have embeds to be a log message
	if m.Content == "" && len(m.Embeds) > 0 && len(m.Attachments) == 0 {
		for _, name := range ignoreBotNames {
			if m.Author.Username == name {
				common.Log.Debugf("Ignoring webhook message by %v", m.Author.Tag())
				return nil, nil
			}
		}
	}

	// give some time for PK to process the message
	time.Sleep(2 * time.Second)

	// check if we handled this message already
	if bot.HandledMessages.Exists(m.ID) {
		bot.HandledMessages.Remove(m.ID)
		return nil, nil
	}

	common.Log.Debugf("No PK info for webhook message %v, falling back to API", m.ID)

	pkm, err := pk.Message(pkgo.Snowflake(m.ID))
	if err != nil {
		if v, ok := err.(*pkgo.PKAPIError); ok {
			if v.Code == pkgo.MessageNotFound {
				return nil, nil
			}
		}

		common.Log.Errorf("Error getting message info from the PK API: %v", err)
		return nil, err
	}

	bot.ProxiedTriggers.Add(discord.MessageID(pkm.Original))

	if pkm.System == nil || pkm.Member == nil {
		err = bot.DB.UpdateUserID(m.ID, discord.UserID(pkm.Sender))
	} else {
		err = bot.DB.UpdatePKInfo(m.ID, pkm.Sender, pkm.System.ID, pkm.Member.ID)
	}
	return nil, err
}

var pkBotsToCheck = []discord.UserID{466378653216014359}

var (
	linkRegex   = regexp.MustCompile(`^https:\/\/discord.com\/channels\/\d+\/(\d+)\/\d+$`)
	footerRegex = regexp.MustCompile(`^System ID: (\w{5,6}) \| Member ID: (\w{5,6}) \| Sender: .+ \((\d+)\) \| Message ID: (\d+) \| Original Message ID: (\d+)$`)
)

func (bot *Bot) pkMessageCreate(m *gateway.MessageCreateEvent) (resp *handler.Response, err error) {
	// ensure we've actually stored the message
	time.Sleep(500 * time.Millisecond)

	// only handle events that are *probably* a log message
	if len(m.Embeds) == 0 || !linkRegex.MatchString(m.Content) {
		return
	}
	if m.Embeds[0].Footer == nil {
		return
	}
	if !footerRegex.MatchString(m.Embeds[0].Footer.Text) {
		return
	}

	groups := footerRegex.FindStringSubmatch(m.Embeds[0].Footer.Text)

	var (
		sysID    = groups[1]
		memberID = groups[2]
		userID   discord.UserID
		msgID    discord.MessageID
	)

	{
		sf, _ := discord.ParseSnowflake(groups[3])
		userID = discord.UserID(sf)
		sf, _ = discord.ParseSnowflake(groups[4])
		msgID = discord.MessageID(sf)

		originalMessageID, _ := discord.ParseSnowflake(groups[5])
		bot.ProxiedTriggers.Add(discord.MessageID(originalMessageID))
	}

	err = bot.DB.UpdatePKInfo(msgID, pkgo.Snowflake(userID), sysID, memberID)
	if err != nil {
		return nil, err
	}

	bot.HandledMessages.Add(msgID)

	return nil, nil
}
