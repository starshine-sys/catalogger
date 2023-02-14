package config

import (
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type Bot struct {
	*bot.Bot
}

func Setup(root *bot.Bot) {
	log.Debug("Adding config commands")

	bot := &Bot{Bot: root}

	bot.Router.Command("config/channels").Exec(bot.channelsEntry)
}
