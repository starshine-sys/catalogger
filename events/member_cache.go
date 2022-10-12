package events

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/starshine-sys/catalogger/common"
)

func getctx() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (bot *Bot) requestGuildMembers(g *gateway.GuildCreateEvent) {
	for _, ch := range g.Channels {
		bot.Channels.Set(ch.ID, ch)
	}

	for _, r := range g.Roles {
		bot.Roles.Set(r.ID, r)
	}

	ctx, cancel := getctx()
	defer cancel()

	cached, err := bot.MemberStore.IsGuildCached(ctx, g.ID)
	if err != nil {
		common.Log.Errorf("Error checking if guild %v is already cached: %v", err)
	}

	if cached {
		return
	}

	bot.guildsToChunk.Add(g.ID)
	bot.guildsToFetchInvites.Add(g.ID)
}

func (bot *Bot) chunkGuildDelete(g *gateway.GuildDeleteEvent) {
	if g.Unavailable {
		return
	}

	bot.guildsToChunk.Remove(g.ID)
	bot.guildsToFetchInvites.Remove(g.ID)
}

func (bot *Bot) guildMemberChunk(g *gateway.GuildMembersChunkEvent) {
	ctx, cancel := getctx()
	defer cancel()

	err := bot.MemberStore.SetMembers(ctx, g.GuildID, g.Members)
	if err != nil {
		common.Log.Errorf("Error setting members in cache: %v", err)
	}
}

const wsTimeout = 25 * time.Second

func (bot *Bot) chunkGuilds() {
	// tick every 3 seconds to avoid gateway rate limits
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()

	t := time.Now().UTC()

	for range tick.C {
		if bot.guildsToChunk.Length() == 0 && bot.guildsToFetchInvites.Length() == 0 {
			if !bot.doneChunking {
				common.Log.Infof("Done chunking in %v!", time.Since(t).Round(time.Millisecond))
				bot.doneChunking = true
			}
		} else if bot.doneChunking {
			common.Log.Infof("Chunking was finished, but joined new guilds, chunking those")
			bot.doneChunking = false
			t = time.Now().UTC()
		}

		var chunkID, inviteID discord.GuildID
		for _, k := range bot.guildsToChunk.Values() {
			chunkID = k
			bot.guildsToChunk.Remove(k)
			break
		}
		for _, k := range bot.guildsToFetchInvites.Values() {
			inviteID = k
			bot.guildsToFetchInvites.Remove(k)
			break
		}

		if chunkID.IsValid() {
			ctx, cancel := context.WithTimeout(context.Background(), wsTimeout)
			err := bot.State(chunkID).Gateway().Send(ctx, &gateway.RequestGuildMembersCommand{
				GuildIDs: []discord.GuildID{chunkID},
				Query:    option.NewString(""),
			})
			if err != nil {
				cancel()
				common.Log.Errorf("Error chunking members for guild %v: %v", chunkID, err)

				bot.guildsToChunk.Add(chunkID)
			} else {
				cancel()

				ctx, cancel := getctx()

				err = bot.MemberStore.MarkGuildCached(ctx, chunkID)
				if err != nil {
					common.Log.Errorf("Error marking guild as cached: %v", err)
				}

				// we can't defer this as it's an infinite loop
				// so call cancel() manually at the end
				cancel()
			}
		}

		if inviteID.IsValid() {
			inv, err := bot.State(inviteID).GuildInvites(inviteID)
			if err != nil {
				common.Log.Errorf("Error getting invites for %v: %v", inviteID, err)
				continue
			}

			ctx, cancel := getctx()

			err = bot.MemberStore.SetInvites(ctx, inviteID, inv)
			if err != nil {
				common.Log.Errorf("Error setting invites in cache: %v", err)
			}

			// same as above, we can't defer this as it's an infinite loop
			cancel()
		}

	}
}
