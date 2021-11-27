package events

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildCreate(g *gateway.GuildCreateEvent) (resp *handler.Response, err error) {
	bot.GuildsMu.Lock()
	bot.Guilds[g.ID] = g.Guild
	bot.GuildsMu.Unlock()

	// if we joined this guild more than one minute ago, return
	if g.Joined.Time().UTC().Before(time.Now().UTC().Add(-1 * time.Minute)) {
		var exists bool
		if err = bot.DB.QueryRow(context.Background(), "select exists(select id from guilds where id = $1)", g.ID).Scan(&exists); exists {
			if err != nil {
				common.Log.Errorf("Error checking if guild exists in db: %v", err)
			}

			return
		}
	}

	common.Log.Infof("Joined server %v (%v).", g.Name, g.ID)

	if !bot.BotJoinLeaveLog.IsValid() {
		return
	}

	e := discord.Embed{
		Title: "Joined new server",
		Color: bcr.ColourPurple,
		Thumbnail: &discord.EmbedThumbnail{
			URL: g.IconURL(),
		},
		Description: fmt.Sprintf("Joined new server **%v**", g.Name),
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", g.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	return &handler.Response{
		ChannelID: bot.BotJoinLeaveLog,
		Embeds:    []discord.Embed{e},
	}, nil
}
