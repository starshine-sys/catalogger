package events

import (
	"regexp"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/catalogger/db"
)

var botsToCheck = []discord.UserID{466378653216014359}

var (
	linkRegex   = regexp.MustCompile(`^https:\/\/discord.com\/channels\/\d+\/(\d+)\/\d+$`)
	footerRegex = regexp.MustCompile(`^System ID: (\w{5,6}) \| Member ID: (\w{5,6}) \| Sender: .+ \((\d+)\) \| Message ID: (\d+) \| Original Message ID: (\d+)$`)
)

func (bot *Bot) pkMessageCreate(m *gateway.MessageCreateEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "pk_message_create",
			GuildID: m.GuildID,
		}, err)
	}

	if !ch["MESSAGE_DELETE"].IsValid() {
		return
	}

	// only handle PK message events
	var isPK bool
	for _, u := range botsToCheck {
		if m.Author.ID == u {
			isPK = true
			break
		}
	}
	if !isPK {
		return
	}

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
		sysID     = groups[1]
		memberID  = groups[2]
		userID    discord.UserID
		msgID     discord.MessageID
		channelID discord.ChannelID
	)

	{
		sf, _ := discord.ParseSnowflake(groups[3])
		userID = discord.UserID(sf)
		sf, _ = discord.ParseSnowflake(groups[4])
		msgID = discord.MessageID(sf)
		sf, _ = discord.ParseSnowflake(linkRegex.FindStringSubmatch(m.Content)[1])
		channelID = discord.ChannelID(sf)

		originalMessageID, _ := discord.ParseSnowflake(groups[5])
		bot.ProxiedTriggersMu.Lock()
		bot.ProxiedTriggers[discord.MessageID(originalMessageID)] = struct{}{}
		bot.ProxiedTriggersMu.Unlock()
	}

	// get full message
	msg, err := bot.State.Message(channelID, msgID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "pk_message_create",
			GuildID: m.GuildID,
		}, err)
	}

	dbMsg := db.Message{
		MsgID:     msgID,
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  m.GuildID,

		Username: msg.Author.Username,
		Member:   memberID,
		System:   sysID,

		Content: msg.Content,
	}

	err = bot.DB.InsertProxied(dbMsg)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "pk_message_create",
			GuildID: m.GuildID,
		}, err)
	}
}
