package server

import (
	"context"
	"errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/web/proto"
)

// UserGuildList returns a list of guilds with info if they're joined or not.
func (s *RPCServer) UserGuildList(_ context.Context, req *proto.UserGuildListRequest) (resp *proto.UserGuildListResponse, err error) {
	resp = &proto.UserGuildListResponse{}

	for _, i := range req.GetGuildIds() {
		joined := s.guildJoined(discord.GuildID(i))
		resp.Guilds = append(resp.Guilds, &proto.GuildListGuild{
			Id:     i,
			Joined: joined,
		})
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
			Id:       uint64(ch.ID),
			ParentID: uint64(ch.ParentID),
			Name:     ch.Name,
			Position: int32(ch.Position),
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

	_, perms, err := s.guildPerms(g.ID, discord.UserID(req.GetUserId()))
	if err != nil {
		common.Log.Errorf("Error getting permissions for user %v: %v", req.GetUserId(), err)
		return resp, err
	}
	resp.Permissions = uint64(perms)

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

// ClearCache ...
func (s *RPCServer) ClearCache(_ context.Context, req *proto.ClearCacheRequest) (*proto.ClearCacheResponse, error) {
	common.Log.Infof("Clearing cache for %v and channels %v", req.GetGuildId(), req.GetChannelIds())

	guildID := discord.GuildID(req.GetGuildId())
	channelIDs := []discord.ChannelID{}
	for _, id := range req.GetChannelIds() {
		channelIDs = append(channelIDs, discord.ChannelID(id))
	}

	s.clearCache(guildID, channelIDs...)

	return &proto.ClearCacheResponse{Ok: true}, nil
}
