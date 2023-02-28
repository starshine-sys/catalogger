package meta

import (
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type SendData = bot.SendData

type Bot struct {
	*bot.Bot
}

func Setup(root *bot.Bot) {
	log.Debug("Adding meta handlers")

	bot := &Bot{Bot: root}

	bot.AddHandler(
		// shard ready logging
		bot.ready,
		// initializing guild config + logging guild join
		bot.guildCreate,
	)
}
