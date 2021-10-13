package events

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/dustin/go-humanize"
	"github.com/starshine-sys/bcr"
)

// GitVer is the git version (commit hash)
var GitVer = "[unknown]"

func init() {
	if GitVer != "[unknown]" {
		return
	}

	git := exec.Command("git", "rev-parse", "--short", "HEAD")
	// ignoring errors *should* be fine? if there's no output we just fall back to "unknown"
	b, _ := git.Output()
	GitVer = strings.TrimSpace(string(b))
	if GitVer == "" {
		GitVer = "[unknown]"
	}
}

func (bot *Bot) ping(ctx bcr.Contexter) (err error) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	t := time.Now()

	err = ctx.SendX("...")
	if err != nil {
		return err
	}

	latency := time.Since(t).Round(time.Millisecond)

	// this will return 0ms in the first minute after the bot is restarted
	// can't do much about that though
	heartbeat := ctx.Session().Gateway.PacerLoop.EchoBeat.Time().Sub(ctx.Session().Gateway.PacerLoop.SentBeat.Time()).Round(time.Millisecond)

	// message counts! that's all we store anyway
	var msgCount int64
	err = bot.DB.QueryRow(context.Background(), "select count(*) from messages").Scan(&msgCount)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	bot.GuildsMu.Lock()
	guilds := len(bot.Guilds)
	bot.GuildsMu.Unlock()

	// database latency
	t = time.Now()
	bot.DB.Channels(ctx.GetChannel().GuildID)
	dbLatency := time.Since(t).Round(time.Microsecond)

	bot.MembersMu.Lock()
	bot.ChannelsMu.Lock()
	bot.RolesMu.Lock()

	e := discord.Embed{
		Color:     bcr.ColourPurple,
		Footer:    &discord.EmbedFooter{Text: fmt.Sprintf("Version %v", GitVer)},
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
					humanize.Comma(msgCount), humanize.Comma(int64(guilds)),
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

	_, err = ctx.EditOriginal(api.EditInteractionResponseData{
		Content: option.NewNullableString(""),
		Embeds:  &[]discord.Embed{e},
	})
	return err
}
