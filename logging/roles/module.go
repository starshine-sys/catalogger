package roles

import "github.com/starshine-sys/catalogger/v2/bot"

type SendData = bot.SendData

type Bot struct {
	*bot.Bot
}

func Setup(root *bot.Bot) {
	bot := &Bot{Bot: root}

	bot.AddHandler(
		// role create logs
		bot.roleCreate,
		// role update logs
		bot.roleUpdate,
	)
}
