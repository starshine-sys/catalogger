package messages

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type SendData = bot.SendData

type Bot struct {
	*bot.Bot

	// messages that triggered a proxy and should not be logged
	proxyTriggers *common.Set[discord.MessageID]
	// pluralkit messages that already have data from the pk;log channel
	handledMessages *common.Set[discord.MessageID]
}

func Setup(root *bot.Bot) {
	log.Debug("Adding messages handlers")

	bot := &Bot{
		Bot: root,

		proxyTriggers:   common.NewSet[discord.MessageID](),
		handledMessages: common.NewSet[discord.MessageID](),
	}

	ignoreApplications[0] = discord.AppID(bot.Me().ID)

	bot.AddHandler(
		// standard message create handler (handles saving to database)
		bot.messageCreate,
		// pk message create handler (handles adding extra info to proxied messages)
		bot.pkMessageCreate,
		// message delete handler
		bot.messageDelete,
		// message update handler
		bot.messageUpdate,
	)
}
