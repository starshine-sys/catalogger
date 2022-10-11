package redis

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
)

const cachedGuildsKey = "cachedGuilds"

func (s *Store) IsGuildCached(ctx context.Context, guildID discord.GuildID) (bool, error) {
	// we don't actually care about the value, just if it's null
	// if the return value is null the guild isn't cached
	mb := radix.Maybe{}

	err := s.client.Do(ctx, radix.Cmd(&mb, "LPOS", cachedGuildsKey, guildID.String()))
	if err != nil {
		return false, err
	}

	return !mb.Null, nil
}

func (s *Store) MarkGuildCached(ctx context.Context, guildID discord.GuildID) error {
	return s.client.Do(ctx, radix.Cmd(nil, "RPUSH", cachedGuildsKey, guildID.String()))
}
