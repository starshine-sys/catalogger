package db

import (
	"context"
	"database/sql"
	"embed"

	"emperror.dev/errors"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/v2/common/log"

	migrate "github.com/rubenv/sql-migrate"

	// pgx driver for migrations
	_ "github.com/jackc/pgx/v4/stdlib"
)

// sq is a squirrel builder for postgres
var sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type DB struct {
	*pgxpool.Pool

	Redis radix.Client

	aesKey [32]byte
}

func New(postgres, redis, aesKey string, shouldMigrate bool) (*DB, error) {
	err := RunMigrations(postgres)
	if err != nil {
		return nil, errors.Wrap(err, "running migrations")
	}

	pool, err := pgxpool.Connect(context.Background(), postgres)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to postgres")
	}

	redisPool, err := (&radix.PoolConfig{}).New(context.Background(), "tcp", redis)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to redis")
	}

	db := &DB{
		Pool:  pool,
		Redis: redisPool,
	}

	copy(db.aesKey[:], []byte(aesKey))

	return db, nil
}

//go:embed migrations
var fs embed.FS

var migrations = &migrate.EmbedFileSystemMigrationSource{
	FileSystem: fs,
	Root:       "migrations",
}

// RunMigrations runs all of the migrations in migrations/.
func RunMigrations(postgres string) (err error) {
	db, err := sql.Open("pgx", postgres)
	if err != nil {
		return errors.Wrap(err, "opening database")
	}

	// we close this because we end up using pgx's native driver for all other queries.
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return errors.Wrap(err, "pinging database")
	}

	// run migrations!
	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return errors.Wrap(err, "running migrations")
	}

	if n != 0 {
		log.Debugf("Performed %v migrations!", n)
	} else {
		log.Debugf("The database is already fully migrated")
	}
	return nil
}
