package events

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/starshine-sys/catalogger/common"
)

func (bot *Bot) updateStatusLoop(s *state.State) {
	time.Sleep(5 * time.Second)

	t := time.NewTicker(10 * time.Minute)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	time.Sleep(10 * time.Second)

	go bot.updateStatusInner(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		bot.updateStatusInner(ctx)
	}
}

func (bot *Bot) updateStatusInner(ctx context.Context) {
	common.Log.Infof("Updating status and posting stats")

	guildCount := 0
	bot.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		guilds, _ := state.GuildStore.Guilds()
		guildCount += len(guilds)
	})

	status := discord.IdleStatus
	if bot.doneChunking {
		status = discord.OnlineStatus

		go bot.postServerStats(ctx, guildCount)
	} else {
		common.Log.Infof("Not done chunking, setting idle status")
	}

	shardNumber := 0
	bot.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		str := fmt.Sprintf("%vhelp", strings.Split(os.Getenv("PREFIXES"), ",")[0])
		if guildCount != 0 {
			str += fmt.Sprintf(" | in %v servers", guildCount)
		}

		i := shardNumber
		shardNumber++

		go func() {
			i := i
			common.Log.Infof("Setting status for shard #%v", i)
			s := str
			if bot.Router.ShardManager.NumShards() > 1 {
				s = fmt.Sprintf("%v | shard #%v", s, i)
			}

			err := state.Gateway().Send(context.Background(), &gateway.UpdatePresenceCommand{
				Status: status,
				Activities: []discord.Activity{{
					Name: s,
					Type: discord.GameActivity,
				}},
			})
			if err != nil {
				common.Log.Errorf("Error setting status for shard #%v: %v", i, err)
			}
		}()
	})
}

func (bot *Bot) postServerStats(ctx context.Context, count int) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if bot.topGGToken != "" {
		req, err := http.NewRequestWithContext(cctx, "POST", "https://top.gg/api/bots/"+bot.Router.Bot.ID.String()+"/stats", strings.NewReader(`{"server_count":`+strconv.Itoa(count)+`}`))
		if err != nil {
			common.Log.Errorf("Error posting stats to top.gg: %v", req)
			return
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", bot.topGGToken)

		resp, err := bot.client.Do(req)
		if err != nil {
			common.Log.Errorf("Error posting stats to top.gg: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			common.Log.Warnf("Non-200 status code: %v", resp.StatusCode)
		}
	}
}
