package db

import (
	"context"
	"database/sql"
	"embed"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v4/pgxpool"
	migrate "github.com/rubenv/sql-migrate"

	// pgx driver for migrations
	_ "github.com/jackc/pgx/v4/stdlib"

	"go.uber.org/zap"
)

// LongQueryThreshold is the threshold above which queries (not arguments) are logged.
const LongQueryThreshold = 20 * time.Millisecond

//go:embed migrations
var fs embed.FS

// DB ...
type DB struct {
	Pool  *pgxpool.Pool
	Sugar *zap.SugaredLogger

	Hub *sentry.Hub

	// Used for encryption
	AESKey [32]byte

	openConns int32
}

// New ...
func New(url string, sugar *zap.SugaredLogger, hub *sentry.Hub) (*DB, error) {
	log := sugar.Named("db")

	err := runMigrations(url, log)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	db := &DB{
		Pool:  pool,
		Sugar: log,
		Hub:   hub,
	}

	copy(db.AESKey[:], []byte(os.Getenv("AES_KEY")))

	return db, nil
}

func runMigrations(url string, sugar *zap.SugaredLogger) (err error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}

	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: fs,
		Root:       "migrations",
	}

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return err
	}

	if n != 0 {
		sugar.Infof("Performed %v migrations!", n)
	}

	err = db.Close()
	return err
}
