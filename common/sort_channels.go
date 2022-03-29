package common

import (
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
)

// SortChannels sorts the given channels into the order shown in the Discord client.
// It returns a new slice, and does not modify the given slice in place.
func SortChannels(channels []discord.Channel) []discord.Channel {
	channelMap := make(map[discord.ChannelID]discord.Channel, len(channels))
	for _, ch := range channels {
		channelMap[ch.ID] = ch
	}

	var (
		noCategory       = make([]discord.Channel, 0)
		categoryChannels = make([]discord.Channel, 0)
		categories       = make(map[discord.ChannelID][]discord.Channel, 0)
	)
	for _, ch := range channels {
		if ch.Type == discord.GuildCategory {
			categoryChannels = append(categoryChannels, ch)
			continue
		}

		if ch.Type == discord.GuildNewsThread || ch.Type == discord.GuildPrivateThread || ch.Type == discord.GuildPublicThread {
			continue
		}

		if !ch.ParentID.IsValid() {
			noCategory = append(noCategory, ch)
		} else {
			categories[ch.ParentID] = append(categories[ch.ParentID], ch)
		}
	}

	// sort every category individually
	for cat := range categories {
		sort.Slice(categories[cat], func(i, j int) bool {
			return categories[cat][i].Position < categories[cat][j].Position
		})
	}

	// sort categories among each other
	sort.Slice(categoryChannels, func(i, j int) bool {
		return categoryChannels[i].Position < categoryChannels[j].Position
	})

	sorted := make([]discord.Channel, 0, len(channels))
	// add uncategorized channels
	sorted = append(sorted, noCategory...)

	for _, cat := range categoryChannels {
		sorted = append(sorted, cat)
		sorted = append(sorted, categories[cat.ID]...)
	}

	return sorted
}
