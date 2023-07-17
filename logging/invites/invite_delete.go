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

func (bot *Bot) inviteDelete(ev *gateway.InviteDeleteEvent) {
	// update the cached invites when we're done handling this event
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

	// get cached invites, as the event doesn't have much information
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	invs, err := bot.Cabinet.Invites(ctx, ev.GuildID)
	if err != nil {
		log.Errorf("getting cached invites for %v: %v", ev.GuildID, err)
		return
	}

	var (
		inv   discord.Invite
		found bool
	)
	for _, i := range invs {
		if i.Code == ev.Code {
			found = true
			inv = i
			break
		}
	}

	if !found {
		log.Debugf("couldn't find cached invite %q, ignoring event", ev.Code)
		return
	}

	maxUses := fmt.Sprint(inv.MaxUses)
	if inv.MaxUses == 0 {
		maxUses = "Infinite"
	}

	e := discord.Embed{
		Title:       "Invite deleted",
		Color:       common.ColourRed,
		Description: fmt.Sprintf("An invite (**%v**) was deleted in %v.", ev.Code, ev.ChannelID.Mention()),

		Fields: []discord.EmbedField{
			{
				Name:  "Created by",
				Value: fmt.Sprintf("%v\n%v\nID: %v", inv.Inviter.Mention(), inv.Inviter.Tag(), inv.Inviter.ID),
			},
			{
				Name:   "Uses",
				Value:  fmt.Sprint(inv.Uses),
				Inline: true,
			},
			{
				Name:   "Maximum uses",
				Value:  maxUses,
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
