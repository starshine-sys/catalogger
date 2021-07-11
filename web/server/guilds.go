package server

import (
	"context"
	"errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/starshine-sys/catalogger/web/proto"
)

// UserGuildList returns a list of guilds with info if they're joined or not.
func (s *RPCServer) UserGuildList(_ context.Context, req *proto.UserGuildListRequest) (resp *proto.UserGuildListResponse, err error) {
	resp = &proto.UserGuildListResponse{}

	for _, i := range req.GetGuildId() {
		id := discord.GuildID(i)
		s, _ := s.Bot.StateFromGuildID(id)
		if s == nil {
			resp.Guilds = append(resp.Guilds, &proto.GuildListGuild{
				Id:     i,
				Joined: false,
			})
			continue
		} else {
			guilds, err := s.GuildStore.Guilds()
			if err != nil {
				resp.Guilds = append(resp.Guilds, &proto.GuildListGuild{
					Id:     i,
					Joined: false,
				})
				continue
			}

			joined := false
			for _, g := range guilds {
				if g.ID == id {
					joined = true
					break
				}
			}

			resp.Guilds = append(resp.Guilds, &proto.GuildListGuild{
				Id:     i,
				Joined: joined,
			})
		}
	}

	return resp, nil
}

// ErrGuildNotFound ...
var ErrGuildNotFound = errors.New("guild not found")

// Guild gets basic guild info from the server, including a channel list.
func (s *RPCServer) Guild(ctx context.Context, req *proto.GuildRequest) (resp *proto.GuildResponse, err error) {
	id := discord.GuildID(req.GetId())

	st, _ := s.Bot.StateFromGuildID(id)
	if st == nil {
		return nil, ErrGuildNotFound
	}

	g, err := st.Guild(id)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	resp = &proto.GuildResponse{
		Id:   uint64(g.ID),
		Name: g.Name,
		Icon: g.Icon,
	}

	channels, err := st.Channels(id)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	for _, ch := range channels {
		protoCh := &proto.GuildChannel{
			Id:         uint64(ch.ID),
			CategoryId: uint64(ch.CategoryID),
			Name:       ch.Name,
			Position:   int32(ch.Position),
		}

		switch ch.Type {
		case discord.GuildText:
			protoCh.Type = proto.GuildChannel_TEXT
		case discord.GuildCategory:
			protoCh.Type = proto.GuildChannel_CATEGORY
		case discord.GuildVoice, discord.GuildStageVoice:
			protoCh.Type = proto.GuildChannel_VOICE
		case discord.GuildNewsThread, discord.GuildPublicThread, discord.GuildPrivateThread:
			protoCh.Type = proto.GuildChannel_THREAD
		case discord.GuildNews:
			protoCh.Type = proto.GuildChannel_NEWS
		default:
			protoCh.Type = proto.GuildChannel_UNKNOWN
		}

		resp.Channels = append(resp.Channels, protoCh)
	}

	return resp, nil
}

// GuildUserCount gets the guild and user count
func (s *RPCServer) GuildUserCount(ctx context.Context, req *proto.GuildUserCountRequest) (resp *proto.GuildUserCountResponse, err error) {
	resp = &proto.GuildUserCountResponse{}

	s.Bot.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		guilds, err := state.GuildStore.Guilds()
		if err == nil {
			resp.GuildCount = int64(len(guilds))
		}
	})

	resp.UserCount = s.memberCount()

	return resp, nil
}
