package redis

import (
	"context"

	"emperror.dev/errors"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/v2/store"
)

var _ store.MemberStore = (*Store)(nil)

type Store struct {
	client radix.Client
}

func New(url string) (*Store, error) {
	client, err := (&radix.PoolConfig{}).New(context.Background(), "tcp", url)
	if err != nil {
		return nil, errors.Wrap(err, "creating radix client")
	}

	return &Store{client: client}, nil
}
