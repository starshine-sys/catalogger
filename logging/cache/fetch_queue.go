package cache

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

// fetchLoop fetches one guild every [X] seconds on a loop
func (bot *Bot) fetchLoop(s *state.State) {
	// close on interrupt signal
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go bot.fetchOneGuild(s)
		case <-ctx.Done():
			return
		}
	}
}

// fetchOneGuild is ran on a timer to fetch a single guild's information (that is not automatically sent by the gateway)
// This includes the full member list (which is handled in guildMembersChunk) and the list of invites.
func (bot *Bot) fetchOneGuild(s *state.State) {
	shardID := s.Ready().Shard.ShardID()

	bot.guildsMu.Lock()
	var memberFetchID, inviteFetchID discord.GuildID

	// get a single (mostly) random guild ID, for both members and invites
	for k := range bot.guildsToFetchMembers[shardID] {
		memberFetchID = k
		delete(bot.guildsToFetchMembers[shardID], k)
		break
	}

	for k := range bot.guildsToFetchInvites[shardID] {
		inviteFetchID = k
		delete(bot.guildsToFetchInvites[shardID], k)
		break
	}
	// we can't defer this because the function does things with the chosen values.
	// if it's deferred, only one shard can fetch at a time, which we don't want.
	bot.guildsMu.Unlock()

	if memberFetchID.IsValid() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Debugf("requesting members for %v", memberFetchID)

		err := s.Gateway().Send(ctx, &gateway.RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{memberFetchID},
			Query:    option.NewString(""),
			Limit:    0,
		})
		if err != nil {
			log.Errorf("sending chunk request for %v: %v", memberFetchID, err)

			bot.guildsMu.Lock()
			bot.addToMemberFetchQueue(shardID, memberFetchID)
			bot.guildsMu.Unlock()
		}
	}

	if inviteFetchID.IsValid() {
		log.Debugf("getting invite list for %v", inviteFetchID)

		invs, err := bot.Router.Rest.GuildInvites(inviteFetchID)
		if err != nil {
			log.Errorf("getting invite list for %v: %v", inviteFetchID, err)

			if httpErr, ok := err.(*httputil.HTTPError); ok {
				if httpErr.Status == http.StatusForbidden || httpErr.Status == http.StatusUnauthorized {
					log.Debugf("error getting invites for %v is forbidden/unauthorized", inviteFetchID)
					return
				}
			}

			bot.guildsMu.Lock()
			bot.addToInviteFetchQueue(shardID, memberFetchID)
			bot.guildsMu.Unlock()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = bot.Cabinet.SetInvites(ctx, inviteFetchID, invs)
		if err != nil {
			log.Errorf("setting invites for %v: %v", inviteFetchID, err)
		}
	}
}

func (bot *Bot) guildMembersChunk(ev *gateway.GuildMembersChunkEvent) {
	log.Debugf("received chunk %d/%d for guild %v", ev.ChunkIndex, ev.ChunkCount, ev.GuildID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := bot.Cabinet.SetMembers(ctx, ev.GuildID, ev.Members)
	if err != nil {
		log.Errorf("setting members for %v (chunk %d/%d): %v", ev.GuildID, ev.ChunkIndex, ev.ChunkCount, err)
		return
	}
}
