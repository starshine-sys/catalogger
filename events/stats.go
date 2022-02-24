package events

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/dustin/go-humanize"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
)

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
	heartbeat := ctx.Session().Gateway().EchoBeat().Sub(ctx.Session().Gateway().SentBeat()).Round(time.Millisecond)

	// database latency
	t = time.Now()
	_, err = bot.DB.Channels(ctx.GetChannel().GuildID)
	if err != nil {
		// we don't use the value here but good to log this *anyway* just in case
		common.Log.Errorf("Error fetching channels: %v", err)
	}

	dbLatency := time.Since(t).Round(time.Microsecond)
	statsQuery := bot.lastStatsQuery
	if statsQuery >= time.Second {
		statsQuery = statsQuery.Round(time.Millisecond)
	} else {
		statsQuery = statsQuery.Round(time.Microsecond)
	}

	e := discord.Embed{
		Color:     bcr.ColourPurple,
		Footer:    &discord.EmbedFooter{Text: fmt.Sprintf("Version %v (%v on %v/%v)", common.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)},
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
Cached %v channels and %v roles
Last statistics query took %v`,
					humanize.Comma(bot.msgCount), humanize.Comma(bot.guildCount),
					humanize.Comma(bot.channelCount),
					humanize.Comma(bot.roleCount),
					statsQuery.Round(time.Microsecond),
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

func (bot *Bot) counts() (
	guildCount int64,
	channelCount int64,
	roleCount int64,
	msgCount int64,
	timeTaken time.Duration,
) {
	t := time.Now()

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
		common.Log.Errorf("Error getting message count: %v", err)
	}

	bot.lastStatsQuery = time.Since(t)

	return bot.guildCount, bot.channelCount, bot.roleCount, bot.msgCount, bot.lastStatsQuery
}
