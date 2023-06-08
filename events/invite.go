package events

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) inviteCreateEvent(ev *gateway.InviteCreateEvent) (resp *handler.Response, err error) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.InviteCreate].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.InviteCreate],
		GuildID:   ev.GuildID,
	}

	maxUses := fmt.Sprint(ev.MaxUses)
	if ev.MaxUses == 0 {
		maxUses = "Infinite"
	}
	expires := "Never"
	if ev.MaxAge != 0 {
		expires = fmt.Sprintf("<t:%v>", time.Now().UTC().Add(ev.MaxAge.Duration()).Unix())
	}

	resp.Embeds = []discord.Embed{{
		Title:       "Invite created",
		Color:       bcr.ColourGreen,
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
	}}

	return resp, nil
}

func (bot *Bot) inviteDeleteEvent(ev *gateway.InviteDeleteEvent) (resp *handler.Response, err error) {
	var (
		found bool
		inv   discord.Invite
	)

	ctx, cancel := getctx()
	defer cancel()

	is, err := bot.MemberStore.Invites(ctx, ev.GuildID)
	if err != nil {
		common.Log.Errorf("Error fetching invites for %v: %v", ev.GuildID, err)
		return
	}

	for _, i := range is {
		if i.Code == ev.Code {
			found = true
			inv = i
			break
		}
	}

	if !found {
		return
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.InviteDelete].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.InviteDelete],
	}

	maxUses := fmt.Sprint(inv.MaxUses)
	if inv.MaxUses == 0 {
		maxUses = "Infinite"
	}

	resp.Embeds = []discord.Embed{{
		Title:       "Invite deleted",
		Color:       bcr.ColourRed,
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
	}}

	return resp, nil
}
