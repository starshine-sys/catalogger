package events

import (
	"fmt"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) emojiUpdate(ev *gateway.GuildEmojisUpdateEvent) (resp *handler.Response, err error) {
	bot.GuildsMu.Lock()
	guild, ok := bot.Guilds[ev.GuildID]
	if !ok {
		bot.GuildsMu.Unlock()
		return
	}
	bot.GuildsMu.Unlock()

	defer func() {
		bot.GuildsMu.Lock()
		g := bot.Guilds[ev.GuildID]
		g.Emojis = ev.Emojis
		bot.Guilds[ev.GuildID] = g
		bot.GuildsMu.Unlock()
	}()

	var removed, added []discord.Emoji
	for _, oldEmoji := range guild.Emojis {
		if !emojiIn(ev.Emojis, oldEmoji) {
			removed = append(removed, oldEmoji)
		}
	}
	for _, newEmoji := range ev.Emojis {
		if !emojiIn(guild.Emojis, newEmoji) {
			added = append(added, newEmoji)
		}
	}

	var embeds []discord.Embed

	if len(added) == 0 && len(removed) == 0 {
		e := emojiRenameEmbed(guild.Emojis, ev.Emojis)
		if e == nil {
			return
		}
		embeds = append(embeds, *e)
	}

	// not sure if more than one emoji can be changed in a single update event, but just iterate through them all just in case
	for _, em := range added {
		e := discord.Embed{
			Title:       "Emoji created",
			Description: fmt.Sprintf("%v [%v](%v)", em.String(), em.Name, em.EmojiURL()),
			Fields: []discord.EmbedField{{
				Name:  "Animated",
				Value: strconv.FormatBool(em.Animated),
			}},
			Footer: &discord.EmbedFooter{
				Text: "ID: " + em.ID.String(),
			},
			Timestamp: discord.NowTimestamp(),
			Color:     bcr.ColourGreen,
			Thumbnail: &discord.EmbedThumbnail{
				URL: em.EmojiURL(),
			},
		}
		if em.User.ID.IsValid() {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "User",
				Value: fmt.Sprintf("%v (%v)", em.User.Tag(), em.User.ID),
			})
		}

		embeds = append(embeds, e)
	}
	for _, em := range removed {
		embeds = append(embeds, discord.Embed{
			Title:       "Emoji removed",
			Description: fmt.Sprintf("[%v](%v)", em.Name, em.EmojiURL()),
			Fields: []discord.EmbedField{{
				Name:  "Animated",
				Value: strconv.FormatBool(em.Animated),
			}},
			Footer: &discord.EmbedFooter{
				Text: "ID: " + em.ID.String(),
			},
			Timestamp: discord.NowTimestamp(),
			Color:     bcr.ColourRed,
			Thumbnail: &discord.EmbedThumbnail{
				URL: em.EmojiURL(),
			},
		})
	}

	if len(embeds) == 0 {
		return
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildEmojisUpdate].IsValid() {
		return
	}

	if len(embeds) > 10 {
		embeds = embeds[:9]
	}

	return &handler.Response{
		ChannelID: ch[keys.GuildEmojisUpdate],
		GuildID:   ev.GuildID,
		Embeds:    embeds,
	}, nil
}

func emojiRenameEmbed(old, new []discord.Emoji) *discord.Embed {
	for _, old := range old {
		for _, new := range new {
			if old.ID == new.ID && old.Name != new.Name {
				return &discord.Embed{
					Title:       "Emoji renamed",
					Description: fmt.Sprintf("%v %v → [%v](%v)", new.String(), old.Name, new.Name, new.EmojiURL()),
					Footer: &discord.EmbedFooter{
						Text: "ID: " + old.ID.String(),
					},
					Timestamp: discord.NowTimestamp(),
					Color:     bcr.ColourBlue,
					Thumbnail: &discord.EmbedThumbnail{
						URL: new.EmojiURL(),
					},
				}
			}
		}
	}
	return nil
}

func emojiIn(s []discord.Emoji, e discord.Emoji) (exists bool) {
	for _, se := range s {
		if e.ID == se.ID {
			return true
		}
	}
	return false
}
