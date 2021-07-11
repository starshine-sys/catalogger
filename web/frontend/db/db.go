package db

import (
	"context"
	"time"

	"github.com/starshine-sys/catalogger/db"
)

// DB is a database with more site-specific methods
type DB struct {
	*db.DB
}

// Token ...
type Token struct {
	Cookie  string
	Token   []byte
	Expires time.Time
}

func (db *DB) setToken(t Token) (err error) {
	_, err = db.Pool.Exec(context.Background(), "insert into access_tokens (cookie, token, expires) values ($1, $2, $3)", t.Cookie, t.Token, t.Expires)
	return
}
