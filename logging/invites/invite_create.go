package invites

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) inviteCreate(ev *gateway.InviteCreateEvent) {
	defer func() {
		inv, err := bot.Router.Rest.GuildInvites(ev.GuildID)
		if err != nil {
			log.Errorf("getting invites for %v: %v", ev.GuildID, err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = bot.Cabinet.SetInvites(ctx, ev.GuildID, inv)
		if err != nil {
			log.Errorf("setting invites for %v: %v", ev.GuildID, err)
		}
	}()

	if !bot.ShouldLog() {
		return
	}

	maxUses := fmt.Sprint(ev.MaxUses)
	if ev.MaxUses == 0 {
		maxUses = "Infinite"
	}
	expires := "Never"
	if ev.MaxAge != 0 {
		expires = fmt.Sprintf("<t:%v>", time.Now().UTC().Add(ev.MaxAge.Duration()).Unix())
	}

	e := discord.Embed{
		Title:       "Invite created",
		Color:       common.ColourGreen,
		Description: fmt.Sprintf("A new invite (**%v**) was created for %v.", ev.Code, ev.ChannelID.Mention()),

		Fields: []discord.EmbedField{
			{
				Name:  "Created by",
				Value: fmt.Sprintf("%v\n%v\nID: %v", ev.Inviter.Mention(), ev.Inviter.Tag(), ev.Inviter.ID),
			},
			{
				Name:   "Maximum uses",
				Value:  maxUses,
				Inline: true,
			},
			{
				Name:   "Expires",
				Value:  expires,
				Inline: true,
			},
		},

		Footer: &discord.EmbedFooter{
			Text: ev.Code,
		},
		Timestamp: discord.NowTimestamp(),
	}

	bot.Send(ev.GuildID, ev, SendData{
		Embeds: []discord.Embed{e},
	})
}
