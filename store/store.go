// Package store defines interfaces for storing Discord data.
package store

import (
	"context"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
)

const ErrNotFound = errors.Sentinel("value not found in store")

// MemberStore stores members and invites
type MemberStore interface {
	IsGuildCached(ctx context.Context, guildID discord.GuildID) (bool, error)
	MarkGuildCached(ctx context.Context, guildID discord.GuildID) error

	Member(ctx context.Context, guildID discord.GuildID, userID discord.UserID) (discord.Member, error)
	Members(ctx context.Context, guildID discord.GuildID) ([]discord.Member, error)
	SetMember(ctx context.Context, guildID discord.GuildID, m discord.Member) error
	MemberExists(ctx context.Context, guildID discord.GuildID, userID discord.UserID) (bool, error)

	// This can easily just wrap SetMember, this function is separate for optimization reasons
	SetMembers(ctx context.Context, guildID discord.GuildID, ms []discord.Member) error

	DeleteMember(ctx context.Context, guildID discord.GuildID, userID discord.UserID) error

	Invites(ctx context.Context, guildID discord.GuildID) ([]discord.Invite, error)
	SetInvites(ctx context.Context, guildID discord.GuildID, is []discord.Invite) error
}

type ChannelStore interface {
	Channels(context.Context, discord.GuildID) ([]discord.Channel, error)
	Channel(context.Context, discord.ChannelID) (discord.Channel, error)

	SetChannel(context.Context, discord.GuildID, discord.Channel) error
	SetChannels(context.Context, discord.GuildID, []discord.Channel) error
	RemoveChannel(context.Context, discord.GuildID, discord.ChannelID) error
	RemoveChannels(context.Context, discord.GuildID) error
}

type GuildStore interface {
	Guild(context.Context, discord.GuildID) (discord.Guild, error)

	GuildSet(context.Context, discord.Guild) error
	GuildRemove(context.Context, discord.GuildID) error
}

type RoleStore interface {
	Roles(ctx context.Context, guildID discord.GuildID) ([]discord.Role, error)
	Role(ctx context.Context, guildID discord.GuildID, roleID discord.RoleID) (discord.Role, error)
	SetRoles(ctx context.Context, guildID discord.GuildID, rs []discord.Role) error
	SetRole(ctx context.Context, guildID discord.GuildID, r discord.Role) error
	RemoveRole(ctx context.Context, guildID discord.GuildID, roleID discord.RoleID) error
	RemoveRoles(ctx context.Context, guildID discord.GuildID) error
}

// Cabinet combines all stores into a single struct.
// As this struct is entirely made up of interfaces, it can be copied around.
type Cabinet struct {
	MemberStore
	ChannelStore
	GuildStore
	RoleStore
}
