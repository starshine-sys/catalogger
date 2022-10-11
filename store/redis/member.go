package redis

import (
	"context"
	"encoding/json"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/v2/store"
)

func guildMemberKey(guildID discord.GuildID) string {
	return "guildMembers:" + guildID.String()
}

func (s *Store) SetMember(ctx context.Context, guildID discord.GuildID, m discord.Member) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return s.client.Do(ctx, radix.Cmd(nil, "HSET", guildMemberKey(guildID), m.User.ID.String(), string(b)))
}

func (s *Store) SetMembers(ctx context.Context, guildID discord.GuildID, ms []discord.Member) error {
	args := make([]string, 0, len(ms)*2+1)

	args = append(args, guildMemberKey(guildID))

	for _, m := range ms {
		args = append(args, m.User.ID.String())

		b, err := json.Marshal(m)
		if err != nil {
			return err
		}

		args = append(args, string(b))
	}

	return s.client.Do(ctx, radix.Cmd(nil, "HSET", args...))
}

func (s *Store) DeleteMember(ctx context.Context, guildID discord.GuildID, userID discord.UserID) error {
	return s.client.Do(ctx, radix.Cmd(nil, "HDEL", guildMemberKey(guildID), userID.String()))
}

func (s *Store) Member(ctx context.Context, guildID discord.GuildID, userID discord.UserID) (m discord.Member, err error) {
	var raw []byte

	err = s.client.Do(ctx, radix.Cmd(&raw, "HGET", guildMemberKey(guildID), userID.String()))
	if err != nil {
		return m, err
	}

	if raw == nil {
		return m, store.ErrNotFound
	}

	return m, json.Unmarshal(raw, &m)
}

func (s *Store) Members(ctx context.Context, guildID discord.GuildID) (ms []discord.Member, err error) {
	mmap := map[string][]byte{}

	err = s.client.Do(ctx, radix.Cmd(&mmap, "HGETALL", guildMemberKey(guildID)))
	if err != nil {
		return nil, err
	}

	ms = make([]discord.Member, 0, len(mmap))
	for _, src := range mmap {
		var m discord.Member
		err = json.Unmarshal(src, &m)
		if err != nil {
			return nil, err
		}

		ms = append(ms, m)
	}

	return ms, nil
}

func (s *Store) MemberExists(ctx context.Context, guildID discord.GuildID, userID discord.UserID) (bool, error) {
	var i int
	// HEXISTS returns 0 if the field was not found, 1 if it exists
	err := s.client.Do(ctx, radix.Cmd(&i, "HEXISTS", guildMemberKey(guildID), userID.String()))
	if err != nil {
		return false, err
	}
	return i == 1, nil
}
