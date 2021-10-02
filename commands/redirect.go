package commands

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) redirect(ctx *bcr.Context) (err error) {
	conn, err := bot.DB.Obtain()
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	defer conn.Release()

	if len(ctx.Args) == 0 {
		m, err := bot.DB.Redirects(conn, ctx.Message.GuildID)
		if err != nil {
			return bot.DB.ReportCtx(ctx, err)
		}

		if len(m) == 0 {
			_, err = ctx.Send("No channels are having their logs redirected.")
			return err
		}

		var s []string
		for k, v := range m {
			s = append(s, fmt.Sprintf("- <#%v> logging to <#%v>\n", k, v))
		}

		_, _, err = ctx.ButtonPages(
			bcr.StringPaginator("Channel log redirects", bcr.ColourPurple, s, 10), 15*time.Minute,
		)
		return err
	}

	if len(ctx.Args) < 2 {
		_, err = ctx.Send("You must give both a source and destination channel.")
		return
	}

	m, err := bot.DB.Redirects(conn, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	src, err := ctx.ParseChannel(ctx.Args[0])
	if err != nil || src.GuildID != ctx.Message.GuildID {
		_, err = ctx.Send("Source channel not found.")
		return
	}

	var dest discord.ChannelID
	if ctx.Args[1] == "-clear" || ctx.Args[1] == "--clear" || ctx.Args[1] == "clear" {
		dest = 0
	} else {
		destCh, err := ctx.ParseChannel(ctx.Args[1])
		if err != nil || destCh.GuildID != ctx.Message.GuildID || destCh.Type != discord.GuildText {
			_, err = ctx.Send("Destination channel not found.")
			return err
		}

		dest = destCh.ID
	}

	if dest == 0 {
		delete(m, src.ID.String())
	} else {
		m[src.ID.String()] = dest
	}

	err = bot.DB.SetRedirects(conn, ctx.Message.GuildID, m)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	s := fmt.Sprintf("Events from %v are now logging to %v.", src.Mention(), dest.Mention())
	if dest == 0 {
		s = fmt.Sprintf("Events from %v are now logging to the default log channel.", src.Mention())
	}

	_, err = ctx.Reply(s)
	return
}
