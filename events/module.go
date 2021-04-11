package events

import (
	"context"
	"os"
	"sync"
	"time"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/handler"
	"github.com/starshine-sys/bcr"
	"go.uber.org/zap"
)

// Bot ...
type Bot struct {
	*bcr.Router

	DB    *db.DB
	Sugar *zap.SugaredLogger

	ProxiedTriggers   map[discord.MessageID]struct{}
	ProxiedTriggersMu sync.Mutex

	Invites  map[discord.GuildID][]discord.Invite
	InviteMu sync.Mutex

	MessageDeleteCache  *ttlcache.Cache
	MessageUpdateCache  *ttlcache.Cache
	GuildMemberAddCache *ttlcache.Cache

	BotJoinLeaveLog discord.ChannelID
}

// Init ...
func Init(r *bcr.Router, db *db.DB, s *zap.SugaredLogger) {
	joinLeaveLog, _ := discord.ParseSnowflake(os.Getenv("JOIN_LEAVE_LOG"))

	b := &Bot{
		Router: r,
		DB:     db,
		Sugar:  s,

		ProxiedTriggers: map[discord.MessageID]struct{}{},
		Invites:         map[discord.GuildID][]discord.Invite{},

		MessageDeleteCache:  ttlcache.NewCache(),
		MessageUpdateCache:  ttlcache.NewCache(),
		GuildMemberAddCache: ttlcache.NewCache(),

		BotJoinLeaveLog: discord.ChannelID(joinLeaveLog),
	}
	b.MessageDeleteCache.SetTTL(10 * time.Minute)
	b.MessageUpdateCache.SetTTL(10 * time.Minute)
	b.GuildMemberAddCache.SetTTL(10 * time.Minute)

	// add join/leave log handlers
	b.Router.State.PreHandler = handler.New()
	b.Router.State.PreHandler.Synchronous = true
	b.Router.State.PreHandler.AddHandler(b.guildDelete)
	b.Router.State.AddHandler(b.guildCreate)

	// add guild create handler
	b.State.AddHandler(b.DB.CreateServerIfNotExists)

	// add pluralkit message create/delete handlers
	b.State.AddHandler(b.pkMessageCreate)
	b.State.AddHandler(b.pkMessageCreateFallback)
	b.State.AddHandler(b.pkMessageDelete)

	// add message create/update/delete handlers
	b.State.AddHandler(b.messageCreate)
	b.State.AddHandler(b.messageUpdate)
	b.State.AddHandler(b.messageDelete)

	// add guild member add handlers
	b.State.AddHandler(b.guildMemberAdd)
	b.State.AddHandler(b.invitesReady)
	b.State.AddHandler(b.inviteCreate)
	b.State.AddHandler(b.inviteDelete)
}

func (bot *Bot) cleanMessages() {
	for {
		c, err := bot.DB.Pool.Exec(context.Background(), "delete from pk_messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(-720*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if n := c.RowsAffected(); n != 0 {
			bot.Sugar.Debugf("Deleted %v PK messages older than 30 days.", n)
		}

		c, err = bot.DB.Pool.Exec(context.Background(), "delete from messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(-720*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if n := c.RowsAffected(); n != 0 {
			bot.Sugar.Debugf("Deleted %v messages older than 30 days.", n)
		}

		time.Sleep(1 * time.Minute)
	}
}
