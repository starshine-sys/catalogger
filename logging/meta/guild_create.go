package meta

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) guildCreate(ev *gateway.GuildCreateEvent) {
	exists, err := bot.DB.CreateGuild(ev.ID)
	if err != nil {
		log.Errorf("creating guild %v (%v) in database: %v", ev.ID, ev.Name, err)
	}

	// guild config isn't deleted when catalogger leaves the guild
	// so if the guild already has a configuration, we check when we joined
	// if it's more than a minute ago it's safe to assume we were already in the guild
	if exists && ev.Joined.Time().Before(time.Now().Add(-time.Minute)) {
		return
	}

	log.Infof("Joined new guild %v (%v)", ev.ID, ev.Name)

	if !bot.ShouldLog() {
		return
	}

	bot.Send(discord.NullGuildID, ev, SendData{
		ChannelID: bot.Config.Bot.JoinLeaveLog,
		Embeds: []discord.Embed{{
			Title:       "Joined guild",
			Description: fmt.Sprintf("Joined guild **%v**", ev.Name),
			Color:       common.ColourGreen,
			Timestamp:   ev.Joined,
			Footer: &discord.EmbedFooter{
				Text: "ID: " + ev.ID.String(),
			},
		}},
	})
}
