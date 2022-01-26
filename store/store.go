// Package store defines interfaces for a persistent data store for members and invites.
// Reasoning: these aren't sent in ready/guild create events, so we need to fetch them from Discord, but this means we aren't actually "ready" after a restart until *everything* is fetched.
// Moving it out of the bot means these caches can survive bot restarts (maybe even server restarts if the cache itself is persistent, i.e. redis written to disk)
package store

import (
	"context"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
)

const ErrNotFound = errors.Sentinel("value not found in store")

type Store interface {
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
