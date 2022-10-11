package memory

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/store"
)

var _ store.ChannelStore = (*Store)(nil)

func (s *Store) Channels(_ context.Context, guildID discord.GuildID) (chs []discord.Channel, err error) {
	s.channelsMu.RLock()
	defer s.channelsMu.RUnlock()

	ids, ok := s.guildChannels[guildID]
	if !ok || len(ids) == 0 {
		return nil, store.ErrNotFound
	}

	for _, id := range s.guildChannels[guildID] {
		ch, ok := s.channels[id]
		if ok {
			chs = append(chs, *ch)
		}
	}

	return chs, nil
}

func (s *Store) Channel(_ context.Context, channelID discord.ChannelID) (discord.Channel, error) {
	s.channelsMu.RLock()
	defer s.channelsMu.RUnlock()

	ch, ok := s.channels[channelID]
	if !ok {
		return discord.Channel{}, store.ErrNotFound
	}

	return *ch, nil
}

func (s *Store) SetChannel(_ context.Context, guildID discord.GuildID, ch discord.Channel) error {
	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()

	s.channels[ch.ID] = &ch

	if !contains(s.guildChannels[guildID], ch.ID) {
		s.guildChannels[guildID] = append(s.guildChannels[guildID], ch.ID)
	}

	return nil
}

func (s *Store) SetChannels(_ context.Context, guildID discord.GuildID, chs []discord.Channel) error {
	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()

	for _, ch := range chs {
		s.channels[ch.ID] = &ch

		if !contains(s.guildChannels[guildID], ch.ID) {
			s.guildChannels[guildID] = append(s.guildChannels[guildID], ch.ID)
		}
	}

	return nil
}

func (s *Store) RemoveChannel(ctx context.Context, guildID discord.GuildID, channelID discord.ChannelID) error {
	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()

	delete(s.channels, channelID)
	remove(s.guildChannels[guildID], channelID)

	return nil
}

func (s *Store) RemoveChannels(ctx context.Context, guildID discord.GuildID) error {
	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()

	for _, ch := range s.guildChannels[guildID] {
		delete(s.channels, ch)
	}

	delete(s.guildChannels, guildID)

	return nil
}
