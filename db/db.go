package db

import (
	"context"

	"emperror.dev/errors"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/mediocregopher/radix/v4"
)

// sq is a squirrel builder for postgres
var sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type DB struct {
	*pgxpool.Pool

	Redis radix.Client
}

func New(postgres, redis string) (*DB, error) {
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

	return db, nil
}
