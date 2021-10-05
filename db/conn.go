package db

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"go.uber.org/zap"
)

// Querier is any object that can query the database.
type Querier interface {
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
}

var _ Querier = (*Conn)(nil)
var _ pgxscan.Querier = (*Conn)(nil)

// Conn is a wrapped *pgxpool.Conn.
type Conn struct {
	conn *pgxpool.Conn
	Log  *zap.SugaredLogger

	StartTime time.Time
	ConnID    uuid.UUID

	queries   int32
	openConns *int32
	hasClosed bool

	timer *time.Timer
	tmu   sync.Mutex
}

// Obtain obtains a *Conn from the database.
func (db *DB) Obtain() (*Conn, error) {
	return db.ObtainCtx(context.Background())
}

// ObtainCtx obtains a *Conn from the database.
func (db *DB) ObtainCtx(ctx context.Context) (*Conn, error) {
	pgconn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	atomic.AddInt32(&db.openConns, 1)

	id := uuid.New()

	db.Sugar.Infof("Opened connection %s. Current open connections: %d", id, atomic.LoadInt32(&db.openConns))

	conn := &Conn{
		conn:      pgconn,
		Log:       db.Sugar,
		ConnID:    id,
		StartTime: time.Now(),
		openConns: &db.openConns,
	}
	conn.resetTimer()

	return conn, nil
}

// Release releases the connection.
func (c *Conn) Release() {
	if c.hasClosed {
		return
	}

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}

	atomic.AddInt32(c.openConns, -1)

	c.Log.Infof("Releasing connection %s, open for %s with %d queries. Open connections: %d", aurora.Yellow(c.ConnID), aurora.Green(time.Now().Sub(c.StartTime).Round(time.Millisecond)), aurora.Blue(atomic.LoadInt32(&c.queries)), aurora.Blue(atomic.LoadInt32(c.openConns)))

	c.conn.Release()
}

func (c *Conn) resetTimer() {
	c.tmu.Lock()
	defer c.tmu.Unlock()

	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = time.AfterFunc(10*time.Second, func() {
		c.Log.Warnf("Connection %s has been idle for more than 10 seconds!", c.ConnID)
	})
}

// Query queries the database and returns a pgx.Rows.
func (c *Conn) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	c.resetTimer()

	atomic.AddInt32(&c.queries, 1)

	t := time.Now()

	rows, err := c.conn.Query(ctx, query, args...)

	if time.Since(t) > LongQueryThreshold {
		c.Log.Warnf("Query %s on connection %s took %s", aurora.Blue(query), aurora.Yellow(c.ConnID), aurora.Green(time.Since(t).Round(time.Microsecond)))
	}
	return rows, err
}

// QueryRow queries the database and returns a pgx.Row
func (c *Conn) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	c.resetTimer()

	atomic.AddInt32(&c.queries, 1)

	t := time.Now()

	row := c.conn.QueryRow(ctx, query, args...)

	if time.Since(t) > LongQueryThreshold {
		c.Log.Warnf("Query %s on connection %s took %s", aurora.Blue(query), aurora.Yellow(c.ConnID), aurora.Green(time.Since(t).Round(time.Microsecond)))
	}

	return row
}

// Exec executes a query on the database.
func (c *Conn) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	c.resetTimer()

	atomic.AddInt32(&c.queries, 1)

	t := time.Now()

	ct, err := c.conn.Exec(ctx, query, args...)

	if time.Since(t) > LongQueryThreshold {
		c.Log.Warnf("Query %s on connection %s took %s", aurora.Blue(query), aurora.Yellow(c.ConnID), aurora.Green(time.Since(t).Round(time.Microsecond)))
	}

	return ct, err
}
