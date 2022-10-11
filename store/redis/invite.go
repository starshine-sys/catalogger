package redis

import (
	"context"
	"encoding/json"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/v2/store"
)

func invitesKey(guildID discord.GuildID) string {
	return "guildInvites:" + guildID.String()
}

func (s *Store) Invites(ctx context.Context, guildID discord.GuildID) (is []discord.Invite, err error) {
	var raw []byte

	err = s.client.Do(ctx, radix.Cmd(&raw, "GET", invitesKey(guildID)))
	if err != nil {
		return is, err
	}

	if raw == nil {
		return is, store.ErrNotFound
	}

	return is, json.Unmarshal(raw, &is)
}

func (s *Store) SetInvites(ctx context.Context, guildID discord.GuildID, is []discord.Invite) error {
	b, err := json.Marshal(is)
	if err != nil {
		return err
	}

	return s.client.Do(ctx, radix.Cmd(nil, "SET", invitesKey(guildID), string(b)))
}
