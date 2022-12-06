package redis

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
)

const cachedGuildsKey = "cachedGuilds"

func (s *Store) IsGuildCached(ctx context.Context, guildID discord.GuildID) (bool, error) {
	var resp int
	err := s.client.Do(ctx, radix.Cmd(&resp, "SISMEMBER", cachedGuildsKey, guildID.String()))
	if err != nil {
		return false, err
	}

	return resp == 1, nil
}

func (s *Store) MarkGuildCached(ctx context.Context, guildID discord.GuildID) error {
	return s.client.Do(ctx, radix.Cmd(nil, "SADD", cachedGuildsKey, guildID.String()))
}
