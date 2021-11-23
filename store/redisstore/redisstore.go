package redisstore

import (
	"context"

	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/store"
)

type Store struct {
	client radix.Client
}

var _ store.Store = (*Store)(nil)

func NewStore(url string) (*Store, error) {
	s := &Store{}

	client, err := (&radix.PoolConfig{}).New(context.Background(), "tcp", url)
	if err != nil {
		return nil, err
	}

	s.client = client
	return s, nil
}
