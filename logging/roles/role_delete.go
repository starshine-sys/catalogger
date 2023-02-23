package roles

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/duration"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) roleDelete(ev *gateway.GuildRoleDeleteEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// get previous version of role
	old, err := bot.Cabinet.Role(ctx, ev.GuildID, ev.RoleID)
	if err != nil {
		log.Errorf("getting role %v in guild %v: %v", ev.RoleID, ev.GuildID, err)
		return
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := bot.Cabinet.RemoveRole(ctx, ev.GuildID, ev.RoleID)
		if err != nil {
			log.Errorf("deleting role %v in %v: %v", ev.RoleID, ev.GuildID, err)
		}
	}()

	if !bot.ShouldLog() {
		return
	}

	e := discord.Embed{
		Title: fmt.Sprintf(`Role %q deleted`, old.Name),
		Color: common.ColourRed,
		Fields: []discord.EmbedField{
			{
				Name:  "Created",
				Value: fmt.Sprintf("<t:%v>\n%v", old.ID.Time().Unix(), duration.FormatTime(old.ID.Time())),
			},
			{
				Name:  "Colour",
				Value: old.Color.String(),
			},
		},

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.RoleID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	bot.Send(ev.GuildID, ev, SendData{
		Embeds: []discord.Embed{e},
	})
}
