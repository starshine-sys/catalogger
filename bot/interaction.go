package bot

import (
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) interactionCreate(ev *gateway.InteractionCreateEvent) {
	err := bot.Router.Execute(ev)
	if err != nil {
		log.Errorf("handling interaction %v: %v", ev.ID, err)
	}
}
