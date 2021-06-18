package db

import (
	"context"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// DB ...
type DB struct {
	Pool  *pgxpool.Pool
	Sugar *zap.SugaredLogger

	Hub *sentry.Hub

	// Used for encryption
	AESKey [32]byte
}

// New ...
func New(url string, sugar *zap.SugaredLogger, hub *sentry.Hub) (*DB, error) {

	pool, err := pgxpool.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	db := &DB{
		Pool:  pool,
		Sugar: sugar,
		Hub:   hub,
	}

	copy(db.AESKey[:], []byte(os.Getenv("AES_KEY")))

	return db, nil
}
