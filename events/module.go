package events

import (
	"context"
	"time"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/starshine-sys/bcr"
	"go.uber.org/zap"
)

// Bot ...
type Bot struct {
	*bcr.Router

	DB    *db.DB
	Sugar *zap.SugaredLogger

	MessageDeleteCache  *ttlcache.Cache
	MessageUpdateCache  *ttlcache.Cache
	GuildMemberAddCache *ttlcache.Cache
}

// Init ...
func Init(r *bcr.Router, db *db.DB, s *zap.SugaredLogger) {
	b := &Bot{
		Router:              r,
		DB:                  db,
		Sugar:               s,
		MessageDeleteCache:  ttlcache.NewCache(),
		MessageUpdateCache:  ttlcache.NewCache(),
		GuildMemberAddCache: ttlcache.NewCache(),
	}
	b.MessageDeleteCache.SetTTL(10 * time.Minute)
	b.MessageUpdateCache.SetTTL(10 * time.Minute)
	b.GuildMemberAddCache.SetTTL(10 * time.Minute)

	// add guild create handler
	b.State.AddHandler(b.DB.CreateServerIfNotExists)

	// add pluralkit message create/delete handlers
	b.State.AddHandler(b.pkMessageCreate)
	b.State.AddHandler(b.pkMessageCreateFallback)
	b.State.AddHandler(b.pkMessageDelete)
	b.State.AddHandler(b.messageCreate)
	b.State.AddHandler(b.messageUpdate)
	b.State.AddHandler(b.guildMemberAdd)
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
