package events

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

type memberCacheKey struct {
	GuildID discord.GuildID
	UserID  discord.UserID
}

func getctx() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (bot *Bot) requestGuildMembers(g *gateway.GuildCreateEvent) {
	bot.ChannelsMu.Lock()
	for _, ch := range g.Channels {
		bot.Channels[ch.ID] = ch
	}
	bot.ChannelsMu.Unlock()

	bot.RolesMu.Lock()
	for _, r := range g.Roles {
		bot.Roles[r.ID] = r
	}
	bot.RolesMu.Unlock()

	ctx, cancel := getctx()
	defer cancel()

	cached, err := bot.MemberStore.IsGuildCached(ctx, g.ID)
	if err != nil {
		bot.Sugar.Errorf("Error checking if guild %v is already cached: %v", err)
	}

	if cached {
		return
	}

	bot.chunkMu.Lock()
	bot.guildsToChunk[g.ID] = struct{}{}
	bot.guildsToFetchInvites[g.ID] = struct{}{}
	bot.chunkMu.Unlock()
}

func (bot *Bot) chunkGuildDelete(g *gateway.GuildDeleteEvent) {
	if g.Unavailable {
		return
	}

	bot.chunkMu.Lock()
	delete(bot.guildsToChunk, g.ID)
	delete(bot.guildsToFetchInvites, g.ID)
	bot.chunkMu.Unlock()
}

func (bot *Bot) guildMemberChunk(g *gateway.GuildMembersChunkEvent) {
	ctx, cancel := getctx()
	defer cancel()

	err := bot.MemberStore.SetMembers(ctx, g.GuildID, g.Members)
	if err != nil {
		bot.Sugar.Errorf("Error setting members in cache: %v", err)
	}
}

func (bot *Bot) chunkGuilds() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	t := time.Now().UTC()

	for range tick.C {
		bot.chunkMu.Lock()

		if len(bot.guildsToChunk) == 0 && len(bot.guildsToFetchInvites) == 0 {
			if !bot.doneChunking {
				bot.Sugar.Infof("Done chunking in %v!", time.Since(t).Round(time.Millisecond))
				bot.doneChunking = true
			}
		} else if bot.doneChunking {
			bot.Sugar.Infof("Chunking was finished, but joined new guilds, chunking those")
			bot.doneChunking = false
			t = time.Now().UTC()
		}

		var chunkID, inviteID discord.GuildID
		for k := range bot.guildsToChunk {
			chunkID = k
			delete(bot.guildsToChunk, k)
			break
		}
		for k := range bot.guildsToFetchInvites {
			inviteID = k
			delete(bot.guildsToFetchInvites, k)
			break
		}

		bot.chunkMu.Unlock()

		if chunkID.IsValid() {
			err := bot.State(chunkID).Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
				GuildIDs: []discord.GuildID{chunkID},
				Limit:    0,
			})
			if err != nil {
				bot.Sugar.Errorf("Error chunking members for guild %v: %v", chunkID, err)

				bot.chunkMu.Lock()
				bot.guildsToChunk[chunkID] = struct{}{}
				bot.chunkMu.Unlock()
			} else {
				ctx, cancel := getctx()

				err = bot.MemberStore.MarkGuildCached(ctx, chunkID)
				if err != nil {
					bot.Sugar.Errorf("Error marking guild as cached: %v", err)
				}

				// we can't defer this as it's an infinite loop
				// so call cancel() manually at the end
				cancel()
			}
		}

		if inviteID.IsValid() {
			inv, err := bot.State(inviteID).GuildInvites(inviteID)
			if err != nil {
				bot.Sugar.Errorf("Error getting invites for %v: %v", inviteID, err)
				continue
			}

			ctx, cancel := getctx()

			err = bot.MemberStore.SetInvites(ctx, inviteID, inv)
			if err != nil {
				bot.Sugar.Errorf("Error setting invites in cache: %v", err)
			}

			// same as above, we can't defer this as it's an infinite loop
			cancel()
		}

	}
}
