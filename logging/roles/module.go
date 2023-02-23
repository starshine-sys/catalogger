package roles

import (
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type SendData = bot.SendData

type Bot struct {
	*bot.Bot
}

func Setup(root *bot.Bot) {
	log.Debug("Adding roles handlers")

	bot := &Bot{Bot: root}

	bot.AddHandler(
		// role create logs
		bot.roleCreate,
		// role update logs
		bot.roleUpdate,
		// role delete logs
		bot.roleDelete,
	)
}
