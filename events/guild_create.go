package events

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) guildCreate(g *gateway.GuildCreateEvent) {
	bot.GuildsMu.Lock()
	bot.Guilds[g.ID] = g.Guild
	bot.GuildsMu.Unlock()

	// if we joined this guild more than one minute ago, return
	if g.Joined.Time().UTC().Before(time.Now().UTC().Add(-1 * time.Minute)) {
		var exists bool
		if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where id = $1)", g.ID).Scan(&exists); exists {
			return
		}
	}

	bot.Sugar.Infof("Joined server %v (%v).", g.Name, g.ID)

	if !bot.BotJoinLeaveLog.IsValid() {
		return
	}

	owner := g.OwnerID.Mention()
	if o, err := bot.State(g.ID).User(g.OwnerID); err == nil {
		owner = fmt.Sprintf("%v#%v (%v)", o.Username, o.Discriminator, o.Mention())
	}

	e := discord.Embed{
		Title: "Joined new server",
		Color: bcr.ColourPurple,
		Thumbnail: &discord.EmbedThumbnail{
			URL: g.IconURL(),
		},

		Description: fmt.Sprintf("Joined new server **%v**", g.Name),

		Fields: []discord.EmbedField{{
			Name:  "Owner",
			Value: owner,
		}},

		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", g.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	_, err := bot.State(g.ID).SendEmbeds(bot.BotJoinLeaveLog, e)
	if err != nil {
		bot.Sugar.Errorf("Error sending join log message: %v", err)
	}
	return
}
