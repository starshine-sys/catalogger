package meta

import (
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type Bot struct {
	*bot.Bot
}

func Setup(root *bot.Bot) {
	log.Debug("Adding meta commands")

	bot := &Bot{Bot: root}

	bot.Router.Command("catalogger/help").Exec(bot.help)
	bot.Router.Command("catalogger/invite").Exec(bot.invite)
	bot.Router.Command("catalogger/dashboard").Exec(bot.dashboard)
}
