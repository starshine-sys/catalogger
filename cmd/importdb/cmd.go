package importdb

import (
	"context"
	"encoding/base64"
	"time"

	"emperror.dev/errors"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/urfave/cli/v2"
)

// sq is a squirrel builder for postgres
var sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

var Command = &cli.Command{
	Name:   "importdb",
	Usage:  "Import messages and guild settings from an old database version",
	Action: run,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from-db",
			Usage:    "URL of old database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "from-key",
			Usage:    "Encryption key of old database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "to-db",
			Usage:    "URL of new database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "to-key",
			Usage:    "Encryption key of new database",
			Required: true,
		},
	},
}

func run(c *cli.Context) error {
	log.Warn("WARNING! This will delete any data already stored in the target database!")
	log.Warn("Waiting 10 seconds to let you cancel")
	time.Sleep(10 * time.Second)

	oldDB, err := pgx.Connect(c.Context, c.String("from-db"))
	if err != nil {
		log.Errorf("connecting to old DB: %v", err)
		return err
	}

	newDB, err := pgx.Connect(c.Context, c.String("to-db"))
	if err != nil {
		log.Errorf("connecting to new DB: %v", err)
		return err
	}

	var oldKey, newKey [32]byte
	copy(oldKey[:], []byte(c.String("from-key")))

	newKeyBytes, err := base64.RawStdEncoding.DecodeString(c.String("to-key"))
	if err != nil {
		log.Errorf("decoding new crypt key: %v", err)
		return err
	}
	copy(newKey[:], newKeyBytes)

	// START COPY
	err = cleanNewDatabase(c.Context, newDB)
	if err != nil {
		log.Errorf("cleaning up new database: %v", err)
	}

	err = doMigrateMessages(oldDB, newDB, oldKey, newKey)
	if err != nil {
		log.Errorf("migrating messages: %v", err)
	}
	return nil
}

func cleanNewDatabase(ctx context.Context, db *pgx.Conn) (err error) {
	log.Info("cleaning up new database")
	_, err = db.Exec(ctx, "DELETE FROM guilds")
	if err != nil {
		return errors.Wrap(err, "deleting guilds")
	}

	_, err = db.Exec(ctx, "DELETE FROM messages")
	if err != nil {
		return errors.Wrap(err, "deleting messages")
	}

	_, err = db.Exec(ctx, "DELETE FROM ignored_messages")
	if err != nil {
		return errors.Wrap(err, "deleting ignored_messages")
	}

	_, err = db.Exec(ctx, "DELETE FROM invites")
	if err != nil {
		return errors.Wrap(err, "deleting invites")
	}

	_, err = db.Exec(ctx, "DELETE FROM watchlist")
	if err != nil {
		return errors.Wrap(err, "deleting watchlist")
	}

	return nil
}
