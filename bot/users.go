package bot

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
)

// GuildUser returns a user from the given guild.
// If the user is still a member of the guild, it will grab the user from cache.
// Otherwise, it will request the user object from Discord directly.
func (bot *Bot) GuildUser(guildID discord.GuildID, userID discord.UserID) (*discord.User, error) {
	m, err := bot.Cabinet.Member(context.Background(), guildID, userID)
	if err == nil {
		return &m.User, nil
	}

	bot.usersMu.Lock()
	defer bot.usersMu.Unlock()

	if u, ok := bot.users[userID]; ok {
		return u, nil
	}

	u, err := bot.Router.Rest.User(userID)
	if err != nil {
		return nil, err
	}

	bot.users[userID] = u
	return u, nil
}

// User returns a user from the cache, or from Discord's API if the user is not cached.
// NOTE: This method should be used very sparingly! If a guild ID is available, GuildUser should always be used instead.
func (bot *Bot) User(userID discord.UserID) (*discord.User, error) {
	bot.usersMu.Lock()
	defer bot.usersMu.Unlock()

	if u, ok := bot.users[userID]; ok {
		return u, nil
	}

	u, err := bot.Router.Rest.User(userID)
	if err != nil {
		return nil, err
	}

	bot.users[userID] = u
	return u, nil
}
