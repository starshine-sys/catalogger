package events

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/dustin/go-humanize"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) ping(ctx *bcr.Context) (err error) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	t := time.Now()

	m, err := ctx.Send("...", nil)
	if err != nil {
		return err
	}

	latency := time.Since(t).Round(time.Millisecond)

	// this will return 0ms in the first minute after the bot is restarted
	// can't do much about that though
	heartbeat := ctx.State.Gateway.PacerLoop.EchoBeat.Time().Sub(ctx.State.Gateway.PacerLoop.SentBeat.Time()).Round(time.Millisecond)

	// message counts! that's all we store anyway
	var msgs struct {
		Messages   int64
		PKMessages int64
	}

	err = pgxscan.Get(context.Background(), bot.DB.Pool, &msgs, "select (select count(*) from messages) as messages, (select count(*) from pk_messages) as pk_messages")
	if err != nil {
		bot.Sugar.Errorf("Error getting message counts: %v", err)
	}
	guilds, err := ctx.State.Guilds()
	if err != nil {
		bot.Sugar.Errorf("Error getting guilds: %v", err)
	}

	bot.MembersMu.Lock()
	bot.ChannelsMu.Lock()
	bot.RolesMu.Lock()

	e := discord.Embed{
		Color: bcr.ColourPurple,
		Fields: []discord.EmbedField{
			{
				Name:   "Ping",
				Value:  fmt.Sprintf("Heartbeat: %v\nMessage: %v", heartbeat, latency),
				Inline: true,
			},
			{
				Name:   "Memory usage",
				Value:  fmt.Sprintf("%v / %v", humanize.Bytes(stats.Alloc), humanize.Bytes(stats.Sys)),
				Inline: true,
			},
			{
				Name:   "Garbage collected",
				Value:  humanize.Bytes(stats.TotalAlloc),
				Inline: true,
			},
			{
				Name:   "Goroutines",
				Value:  fmt.Sprint(runtime.NumGoroutine()),
				Inline: true,
			},
			{
				Name: "Uptime",
				Value: fmt.Sprintf(
					"%v\n(Since %v)",
					bcr.HumanizeDuration(bcr.DurationPrecisionSeconds, time.Since(bot.Start)),
					bot.Start.Format("Jan _2 2006, 15:04:05 MST"),
				),
				Inline: true,
			},
			{
				Name: "Numbers",
				Value: fmt.Sprintf(
					`%v messages (%v normal, %v proxied) from %v servers
Cached %v members, %v channels, and %v roles`,
					humanize.Comma(msgs.Messages+msgs.PKMessages), humanize.Comma(msgs.Messages), humanize.Comma(msgs.PKMessages), len(guilds),
					humanize.Comma(int64(len(bot.Members))),
					humanize.Comma(int64(len(bot.Channels))),
					humanize.Comma(int64(len(bot.Roles))),
				),
			},
		},
	}

	bot.MembersMu.Unlock()
	bot.ChannelsMu.Unlock()
	bot.RolesMu.Unlock()

	_, err = ctx.Edit(m, "", &e)
	return err
}
