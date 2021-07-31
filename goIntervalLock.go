package gointerlock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

type Locker struct {
	redisConnector *redis.Client
}

func (s *Locker) Lock(ctx context.Context, key string, lockTtl time.Duration) (success bool, err error) {

	if s.redisConnector != nil {

		if key == "" {
			return false, errors.New("`Distributed Jobs should have a unique name!`")
		}

		res, err := s.redisConnector.SetNX(ctx, key, time.Now().String(), lockTtl).Result()
		if err != nil {
			return false, err
		}
		return res, nil
	}
	//true when lock is disabled
	return true, nil
}

func (s *Locker) Unlock(ctx context.Context, key string) error {
	if s.redisConnector != nil {
		return s.redisConnector.Del(ctx, key).Err()
	} else {
		return nil
	}
}
