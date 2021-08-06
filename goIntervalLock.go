package gointerlock

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

var prefix = "goInterLock"

type Locker struct {
	redisConnector *redis.Client
}

func (s *Locker) RedisLock(ctx context.Context, key string, lockTtl time.Duration) (success bool, err error) {

	if s.redisConnector != nil {

		if key == "" {
			return false, errors.New("`Distributed Jobs should have a unique name!`")
		}

		res, err := s.redisConnector.SetNX(ctx, fmt.Sprintf("%s_%s", prefix, key), time.Now().String(), lockTtl).Result()
		if err != nil {
			return false, err
		}
		return res, nil
	}

	return false, errors.New("`No Redis Connection found`")
}

func (s *Locker) RedisUnlock(ctx context.Context, key string) error {
	if s.redisConnector != nil {
		return s.redisConnector.Del(ctx, fmt.Sprintf("%s_%s", prefix, key)).Err()
	} else {
		return nil
	}
}
