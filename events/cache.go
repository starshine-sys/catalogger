package events

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
)

func (bot *Bot) Member(guildID discord.GuildID, userID discord.UserID) (discord.Member, error) {
	ctx, cancel := getctx()
	defer cancel()

	return bot.MemberContext(ctx, guildID, userID)
}

func (bot *Bot) MemberContext(ctx context.Context, guildID discord.GuildID, userID discord.UserID) (discord.Member, error) {
	m, err := bot.MemberStore.Member(ctx, guildID, userID)
	if err == nil {
		return m, nil
	}

	mp, err := bot.State(guildID).Member(guildID, userID)
	if err == nil {
		return *mp, nil
	}
	return m, err
}

func (bot *Bot) Invites(guildID discord.GuildID) ([]discord.Invite, error) {
	ctx, cancel := getctx()
	defer cancel()

	return bot.MemberStore.Invites(ctx, guildID)
}

func (bot *Bot) SetInvites(guildID discord.GuildID, is []discord.Invite) error {
	ctx, cancel := getctx()
	defer cancel()

	return bot.MemberStore.SetInvites(ctx, guildID, is)
}
