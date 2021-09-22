package events

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) inviteCreateEvent(ev *gateway.InviteCreateEvent) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.InviteCreate,
			GuildID: ev.GuildID,
		}, err)
		return
	}

	if !ch[keys.InviteCreate].IsValid() {
		return
	}

	wh, err := bot.webhookCache(keys.InviteCreate, ev.GuildID, ch[keys.InviteCreate])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.InviteCreate,
			GuildID: ev.GuildID,
		}, err)
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
		Color:       bcr.ColourGreen,
		Description: fmt.Sprintf("A new invite (**%v**) was created for %v.", ev.Code, ev.ChannelID.Mention()),

		Fields: []discord.EmbedField{
			{
				Name:  "Created by",
				Value: fmt.Sprintf("%v\n%v#%v\nID: %v", ev.Inviter.Mention(), ev.Inviter.Username, ev.Inviter.Discriminator, ev.Inviter.ID),
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

	err = webhook.FromAPI(wh.ID, wh.Token, bot.State(ev.GuildID).Client).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.InviteCreate,
			GuildID: ev.GuildID,
		}, err)
		return
	}
}

func (bot *Bot) inviteDeleteEvent(ev *gateway.InviteDeleteEvent) {
	var (
		found bool
		inv   discord.Invite
	)
	bot.InviteMu.Lock()
	for _, i := range bot.Invites[ev.GuildID] {
		if i.Code == ev.Code {
			found = true
			inv = i
			break
		}
	}
	bot.InviteMu.Unlock()

	if !found {
		return
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.InviteDelete,
			GuildID: ev.GuildID,
		}, err)
		return
	}

	if !ch[keys.InviteDelete].IsValid() {
		return
	}

	wh, err := bot.webhookCache(keys.InviteDelete, ev.GuildID, ch[keys.InviteDelete])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.InviteDelete,
			GuildID: ev.GuildID,
		}, err)
		return
	}

	maxUses := fmt.Sprint(inv.MaxUses)
	if inv.MaxUses == 0 {
		maxUses = "Infinite"
	}

	e := discord.Embed{
		Title:       "Invite deleted",
		Color:       bcr.ColourRed,
		Description: fmt.Sprintf("An invite (**%v**) was deleted in %v.", ev.Code, ev.ChannelID.Mention()),

		Fields: []discord.EmbedField{
			{
				Name:  "Created by",
				Value: fmt.Sprintf("%v\n%v#%v\nID: %v", inv.Inviter.Mention(), inv.Inviter.Username, inv.Inviter.Discriminator, inv.Inviter.ID),
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

	err = webhook.FromAPI(wh.ID, wh.Token, bot.State(ev.GuildID).Client).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.InviteDelete,
			GuildID: ev.GuildID,
		}, err)
		return
	}
}
