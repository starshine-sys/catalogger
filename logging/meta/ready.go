package meta

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) ready(ev *gateway.ReadyEvent) {
	log.Debugf("Shard %d/%d is ready!", ev.Shard.ShardID(), ev.Shard.NumShards())

	if !bot.ShouldLog() {
		return
	}

	if !bot.Config.Bot.MetaLog.IsValid() {
		return
	}

	bot.Send(discord.NullGuildID, ev, SendData{
		ChannelID: bot.Config.Bot.MetaLog,
		Embeds: []discord.Embed{{
			Title:       "Shard ready",
			Description: fmt.Sprintf("Shard %d/%d is ready", ev.Shard.ShardID(), ev.Shard.NumShards()),
			Color:       common.ColourPurple,
			Timestamp:   discord.NowTimestamp(),
			Footer: &discord.EmbedFooter{
				Text: ev.User.Tag(),
			},
		}},
	})
}
