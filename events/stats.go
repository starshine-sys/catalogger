package events

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
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

	// database latency
	t = time.Now()
	_, err = bot.DB.Channels(ctx.GetChannel().GuildID)
	if err != nil {
		// we don't use the value here but good to log this *anyway* just in case
		bot.Sugar.Errorf("Error fetching channels: %v", err)
	}

	dbLatency := time.Since(t).Round(time.Microsecond)

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
Cached %v channels and %v roles`,
					humanize.Comma(bot.msgCount), humanize.Comma(bot.guildCount),
					humanize.Comma(bot.channelCount),
					humanize.Comma(bot.roleCount),
				),
			},
		},
	}

	_, err = ctx.EditOriginal(api.EditInteractionResponseData{
		Content: option.NewNullableString(""),
		Embeds:  &[]discord.Embed{e},
	})
	return err
}

func (bot *Bot) countsLoop() {
	if bot.DB.Stats != nil {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()

	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ticker.C:
			// submit metrics
			go bot.counts()
		case <-ctx.Done():
			// break if we're shutting down
			ticker.Stop()
			return
		}
	}
}

func (bot *Bot) counts() (guidlCount int64, channelCount int64, roleCount int64, msgCount int64) {
	bot.GuildsMu.Lock()
	bot.guildCount = int64(len(bot.Guilds))
	bot.GuildsMu.Unlock()

	bot.ChannelsMu.Lock()
	bot.channelCount = int64(len(bot.Channels))
	bot.ChannelsMu.Unlock()

	bot.RolesMu.Lock()
	bot.roleCount = int64(len(bot.Roles))
	bot.RolesMu.Unlock()

	err := bot.DB.QueryRow(context.Background(), "select count(*) from messages").Scan(&bot.msgCount)
	if err != nil {
		bot.Sugar.Errorf("Error getting message count: %v", err)
	}
	return bot.guildCount, bot.channelCount, bot.roleCount, bot.msgCount
}
