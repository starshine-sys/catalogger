package events

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/gateway"
)

func (bot *Bot) inviteCreate(g *gateway.InviteCreateEvent) {
	inv, err := bot.State(g.GuildID).GuildInvites(g.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting invites for %v: %v", g.GuildID, err)
		return
	}

	ctx, cancel := getctx()
	defer cancel()

	err = bot.MemberStore.SetInvites(ctx, g.GuildID, inv)
	if err != nil {
		bot.Sugar.Errorf("Error updating invite cache for %v: %v", g.GuildID, err)
	}
}

func (bot *Bot) inviteDelete(g *gateway.InviteDeleteEvent) {
	// wait 1 second so we can log the event
	time.Sleep(time.Second)

	_, err := bot.DB.Exec(context.Background(), "delete from invites where code = $1", g.Code)
	if err != nil {
		bot.Sugar.Errorf("Error deleting invite name: %v", err)
	}

	inv, err := bot.State(g.GuildID).GuildInvites(g.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting invites for %v: %v", g.GuildID, err)
		return
	}

	ctx, cancel := getctx()
	defer cancel()

	err = bot.MemberStore.SetInvites(ctx, g.GuildID, inv)
	if err != nil {
		bot.Sugar.Errorf("Error updating invite cache for %v: %v", g.GuildID, err)
	}
}
