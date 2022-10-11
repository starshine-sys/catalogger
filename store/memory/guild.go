package memory

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/store"
)

var _ store.GuildStore = (*Store)(nil)

func (s *Store) Guild(_ context.Context, id discord.GuildID) (discord.Guild, error) {
	s.guildsMu.RLock()
	defer s.guildsMu.RUnlock()

	g, ok := s.guilds[id]
	if !ok {
		return discord.Guild{}, store.ErrNotFound
	}
	return *g, nil
}

func (s *Store) GuildSet(_ context.Context, g discord.Guild) error {
	s.guildsMu.Lock()
	defer s.guildsMu.Unlock()

	s.guilds[g.ID] = &g
	return nil
}

func (s *Store) GuildRemove(_ context.Context, id discord.GuildID) error {
	s.guildsMu.Lock()
	defer s.guildsMu.Unlock()

	delete(s.guilds, id)
	return nil
}
