package db

import (
	"context"
	"database/sql"
	"embed"
	"os"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db/stats"

	// pgx driver for migrations
	_ "github.com/jackc/pgx/v4/stdlib"
)

//go:embed migrations
var fs embed.FS

// DB ...
type DB struct {
	Pool *pgxpool.Pool

	Hub *sentry.Hub

	// Used for encryption
	AESKey [32]byte

	Stats *stats.Client
}

// New ...
func New(url string, hub *sentry.Hub) (*DB, error) {

	err := runMigrations(url)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	db := &DB{
		Pool: pool,
		Hub:  hub,
	}

	influxURL := os.Getenv("INFLUX_URL")
	influxToken := os.Getenv("INFLUX_TOKEN")
	influxDB := os.Getenv("INFLUX_DB")
	if influxURL != "" && influxToken != "" && influxDB != "" {
		idb := strings.SplitN(influxDB, ":", 2)
		db.Stats = stats.New(influxURL, influxToken, idb[0], idb[1])
	}

	copy(db.AESKey[:], []byte(os.Getenv("AES_KEY")))

	return db, nil
}

func runMigrations(url string) (err error) {
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
		common.Log.Infof("Performed %v migrations!", n)
	}

	err = db.Close()
	return err
}

// QueryRow ...
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	go db.Stats.IncQuery()
	return db.Pool.QueryRow(ctx, sql, args...)
}

// Query ...
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	go db.Stats.IncQuery()
	return db.Pool.Query(ctx, sql, args...)
}

// Exec ...
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	go db.Stats.IncQuery()
	return db.Pool.Exec(ctx, sql, args...)
}
