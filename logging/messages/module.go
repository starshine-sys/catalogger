package messages

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common"
)

type SendData = bot.SendData

type Bot struct {
	*bot.Bot

	proxyTriggers, handledMessages *common.Set[discord.MessageID]
}

func Setup(root *bot.Bot) {
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
	)
}
