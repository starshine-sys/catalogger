package events

import (
	"time"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/starshine-sys/bcr"
	"go.uber.org/zap"
)

// Bot ...
type Bot struct {
	*bcr.Router

	DB    *db.DB
	Sugar *zap.SugaredLogger

	MsgWebhookCache *ttlcache.Cache
}

// Init ...
func Init(r *bcr.Router, db *db.DB, s *zap.SugaredLogger) {
	b := &Bot{
		Router:          r,
		DB:              db,
		Sugar:           s,
		MsgWebhookCache: ttlcache.NewCache(),
	}
	b.MsgWebhookCache.SetTTL(10 * time.Minute)

	// add guild create handler
	b.State.AddHandler(b.DB.CreateServerIfNotExists)

	// add pluralkit message create/delete handlers
	b.State.AddHandler(b.pkMessageCreate)
	b.State.AddHandler(b.pkMessageCreateFallback)
	b.State.AddHandler(b.pkMessageDelete)
}
