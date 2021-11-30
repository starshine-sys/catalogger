package frontend

import (
	"context"

	"github.com/mediocregopher/radix/v4"
)

func (s *server) multiRedis(ctx context.Context, cmds ...radix.Action) error {
	for _, cmd := range cmds {
		err := s.Redis.Do(ctx, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}
