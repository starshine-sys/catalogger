package commands

import (
	"fmt"
	"runtime"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/dustin/go-humanize"
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
		},
	}

	_, err = ctx.Edit(m, "", &e)
	return err
}
