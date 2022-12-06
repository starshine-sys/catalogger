package memory

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/store"
)

var _ store.RoleStore = (*Store)(nil)

func (s *Store) Roles(_ context.Context, guildID discord.GuildID) (rls []discord.Role, err error) {
	s.rolesMu.RLock()
	defer s.rolesMu.RUnlock()

	ids, ok := s.guildRoles[guildID]
	if !ok || len(ids) == 0 {
		return nil, store.ErrNotFound
	}

	for _, id := range s.guildRoles[guildID] {
		r, ok := s.roles[id]
		if ok {
			rls = append(rls, *r)
		}
	}

	return rls, nil
}

func (s *Store) Role(_ context.Context, _ discord.GuildID, roleID discord.RoleID) (discord.Role, error) {
	s.rolesMu.RLock()
	defer s.rolesMu.RUnlock()

	r, ok := s.roles[roleID]
	if !ok {
		return discord.Role{}, store.ErrNotFound
	}

	return *r, nil
}

func (s *Store) SetRole(_ context.Context, guildID discord.GuildID, r discord.Role) error {
	s.rolesMu.Lock()
	defer s.rolesMu.Unlock()

	if !contains(s.guildRoles[guildID], r.ID) {
		s.guildRoles[guildID] = append(s.guildRoles[guildID], r.ID)
	}

	return nil
}

func (s *Store) SetRoles(_ context.Context, guildID discord.GuildID, rls []discord.Role) error {
	s.rolesMu.Lock()
	defer s.rolesMu.Unlock()

	for _, r := range rls {
		r := r
		s.roles[r.ID] = &r

		if !contains(s.guildRoles[guildID], r.ID) {
			s.guildRoles[guildID] = append(s.guildRoles[guildID], r.ID)
		}
	}

	return nil
}

func (s *Store) RemoveRole(ctx context.Context, guildID discord.GuildID, roleID discord.RoleID) error {
	s.rolesMu.Lock()
	defer s.rolesMu.Unlock()

	delete(s.roles, roleID)
	remove(s.guildRoles[guildID], roleID)

	return nil
}

func (s *Store) RemoveRoles(ctx context.Context, guildID discord.GuildID) error {
	s.rolesMu.Lock()
	defer s.rolesMu.Unlock()

	for _, r := range s.guildRoles[guildID] {
		delete(s.roles, r)
	}

	delete(s.guildRoles, guildID)

	return nil
}
