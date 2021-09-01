package events

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize"
	"github.com/starshine-sys/bcr"
)

var gitVer string

func init() {
	git := exec.Command("git", "rev-parse", "--short", "HEAD")
	// ignoring errors *should* be fine? if there's no output we just fall back to "unknown"
	b, _ := git.Output()
	gitVer = strings.TrimSpace(string(b))
	if gitVer == "" {
		gitVer = "[unknown]"
	}
}

func (bot *Bot) ping(ctx *bcr.Context) (err error) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	t := time.Now()

	m, err := ctx.Send("...")
	if err != nil {
		return err
	}

	latency := time.Since(t).Round(time.Millisecond)

	// this will return 0ms in the first minute after the bot is restarted
	// can't do much about that though
	heartbeat := ctx.State.Gateway.PacerLoop.EchoBeat.Time().Sub(ctx.State.Gateway.PacerLoop.SentBeat.Time()).Round(time.Millisecond)

	// message counts! that's all we store anyway
	var msgCount int64
	err = bot.DB.Pool.QueryRow(context.Background(), "select count(*) from messages").Scan(&msgCount)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}
	guilds, err := ctx.State.Guilds()
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	// database latency
	t = time.Now()
	bot.DB.Channels(ctx.Message.GuildID)
	dbLatency := time.Since(t).Round(time.Microsecond)

	bot.MembersMu.Lock()
	bot.ChannelsMu.Lock()
	bot.RolesMu.Lock()

	e := discord.Embed{
		Color:     bcr.ColourPurple,
		Footer:    &discord.EmbedFooter{Text: fmt.Sprintf("Version %v", gitVer)},
		Timestamp: discord.NowTimestamp(),
		Fields: []discord.EmbedField{
			{
				Name:   "Ping",
				Value:  fmt.Sprintf("Heartbeat: %v\nMessage: %v\nDatabase: %v", heartbeat, latency, dbLatency),
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
					"%v\n(Since <t:%v:D> <t:%v:T>)",
					bcr.HumanizeDuration(bcr.DurationPrecisionSeconds, time.Since(bot.Start)),
					bot.Start.Unix(), bot.Start.Unix(),
				),
				Inline: true,
			},
			{
				Name: "Numbers",
				Value: fmt.Sprintf(
					`%v messages from %v servers
Cached %v members, %v channels, and %v roles`,
					humanize.Comma(msgCount), humanize.Comma(int64(len(guilds))),
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

	_, err = ctx.Edit(m, "", true, e)
	return err
}
