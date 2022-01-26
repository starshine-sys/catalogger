package events

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/common"
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

// RootChannel returns the given channel's root channel--either the channel itself, or the parent channel if it's a thread.
func (bot *Bot) RootChannel(guildID discord.GuildID, id discord.ChannelID) (*discord.Channel, error) {
	s := bot.State(guildID)

	ch, err := s.Channel(id)
	if err != nil {
		return nil, err
	}

	if IsThread(ch) {
		return s.Channel(ch.ParentID)
	}

	return ch, nil
}

func IsThread(ch *discord.Channel) bool {
	switch ch.Type {
	case discord.GuildNewsThread, discord.GuildPrivateThread, discord.GuildPublicThread:
		return true
	default:
		return false
	}
}

func (bot *Bot) HandleCache(iface interface{}) {
	time.Sleep(2 * time.Second)

	ctx, cancel := getctx()
	defer cancel()

	switch ev := iface.(type) {
	case *gateway.MessageCreateEvent:
		if !ev.GuildID.IsValid() || ev.WebhookID.IsValid() || ev.Member == nil {
			return
		}
		// otherwise, check if member exists and cache them
		ok, err := bot.MemberStore.MemberExists(ctx, ev.GuildID, ev.Author.ID)
		if err != nil {
			common.Log.Errorf("Error checking if member %v:%v exists: %v", ev.GuildID, ev.Author.ID, err)
			return
		}
		if ok {
			return
		}
		err = bot.MemberStore.SetMember(ctx, ev.GuildID, *ev.Member)
		if err != nil {
			common.Log.Errorf("Error storing member %v:%v: %v", ev.GuildID, ev.Author.ID, err)
		}

	case *gateway.GuildMemberUpdateEvent:
		ok, err := bot.MemberStore.MemberExists(ctx, ev.GuildID, ev.User.ID)
		if err != nil {
			common.Log.Errorf("Error checking if member %v:%v exists: %v", ev.GuildID, ev.User.ID, err)
			return
		}
		if ok {
			return
		}

		s, _ := bot.Router.StateFromGuildID(ev.GuildID)
		m, err := s.Client.Member(ev.GuildID, ev.User.ID)
		if err != nil {
			common.Log.Errorf("Error fetching member %v:%v: %v", ev.GuildID, ev.User.ID, err)
			return
		}

		ev.UpdateMember(m)
		err = bot.MemberStore.SetMember(ctx, ev.GuildID, *m)
		if err != nil {
			common.Log.Errorf("Error storing member %v:%v: %v", ev.GuildID, ev.User.ID, err)
		}
	}
}
