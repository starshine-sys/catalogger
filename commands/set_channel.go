package commands

import "github.com/starshine-sys/bcr"

func (bot *Bot) setChannel(ctx *bcr.Context) (err error) {
	ch, err := bot.DB.Channels(ctx.Message.GuildID)
	if err != nil {
		_, err = ctx.Sendf("Error: %v", err)
		return
	}

	if _, ok := ch[ctx.Args[0]]; !ok {
		_, err = ctx.Sendf("Invalid event given. Use `%vevents` for a list of valid events.", ctx.Prefix)
		return
	}

	ch[ctx.Args[0]] = ctx.Channel.ID

	clear, _ := ctx.Flags.GetBool("clear")
	if clear {
		ch[ctx.Args[0]] = 0
	}

	err = bot.DB.SetChannels(ctx.Message.GuildID, ch)
	if err != nil {
		_, err = ctx.Sendf("Error: %v", err)
		return
	}

	if !clear {
		_, err = ctx.Sendf("Now logging ``%v`` events to this channel (%v).", bcr.EscapeBackticks(ctx.Args[0]), ctx.Channel.Mention())
	} else {
		_, err = ctx.Sendf("No longer logging ``%v`` events.", bcr.EscapeBackticks(ctx.Args[0]))
	}
	return
}
