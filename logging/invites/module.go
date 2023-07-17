package invites

import (
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type SendData = bot.SendData

type Bot struct {
	*bot.Bot
}

func Setup(root *bot.Bot) {
	log.Debug("Adding invites handlers")

	b := &Bot{Bot: root}

	b.AddHandler(
		// invite create handler
		b.inviteCreate,
		// invite delete handler
		b.inviteDelete,
	)
}
