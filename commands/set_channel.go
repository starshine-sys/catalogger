package commands

import (
	"strings"

	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) setChannel(ctx *bcr.Context) (err error) {
	ch, err := bot.DB.Channels(ctx.Message.GuildID)
	if err != nil {
		_, err = ctx.Sendf("Error: %v", err)
		return
	}

	events := strings.Split(
		strings.Join(ctx.Args, " "), ",",
	)

	if len(events) == 1 {
		events = ctx.Args
	}

	clear, _ := ctx.Flags.GetBool("clear")
	if clear {
		ch[ctx.Args[0]] = 0
	}

	for _, e := range events {
		e := strings.ToUpper(strings.TrimSpace(e))

		if _, ok := db.DefaultEventMap[e]; !ok {
			_, err = ctx.Sendf("Invalid event (``%v``) given. Use `%vevents` for a list of valid events.", bcr.EscapeBackticks(e), ctx.Prefix)
			return
		}

		ch[e] = ctx.Channel.ID
		if clear {
			ch[e] = 0
		}
	}

	err = bot.DB.SetChannels(ctx.Message.GuildID, ch)
	if err != nil {
		_, err = ctx.Sendf("Error: %v", err)
		return
	}

	if !clear {
		_, err = ctx.Sendf("Now logging ``%v`` events to this channel (%v).", bcr.EscapeBackticks(strings.Join(events, ", ")), ctx.Channel.Mention())
	} else {
		_, err = ctx.Sendf("No longer logging ``%v`` events.", bcr.EscapeBackticks(strings.Join(events, ", ")))
	}
	if err != nil {
		bot.Sugar.Errorf("Error sending message: %v", err)
	}
	err = bot.Router.GetCommand("clearcache").Command(ctx)
	return
}
