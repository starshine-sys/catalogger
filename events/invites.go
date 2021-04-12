package events

import (
	"time"

	"github.com/diamondburned/arikawa/v2/gateway"
)

func (bot *Bot) invitesReady(g *gateway.GuildCreateEvent) {
	inv, err := bot.State.GuildInvites(g.ID)
	if err != nil {
		bot.Sugar.Errorf("Error getting invites for %v: %v", g.ID, err)
		return
	}

	bot.InviteMu.Lock()
	bot.Invites[g.ID] = inv
	bot.InviteMu.Unlock()
}

func (bot *Bot) inviteCreate(g *gateway.InviteCreateEvent) {
	inv, err := bot.State.GuildInvites(g.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting invites for %v: %v", g.GuildID, err)
		return
	}

	bot.InviteMu.Lock()
	bot.Invites[g.GuildID] = inv
	bot.InviteMu.Unlock()
}

func (bot *Bot) inviteDelete(g *gateway.InviteDeleteEvent) {
	// wait 0.5 seconds so we can log the event
	time.Sleep(500 * time.Millisecond)

	inv, err := bot.State.GuildInvites(g.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting invites for %v: %v", g.GuildID, err)
		return
	}

	bot.InviteMu.Lock()
	bot.Invites[g.GuildID] = inv
	bot.InviteMu.Unlock()
}
