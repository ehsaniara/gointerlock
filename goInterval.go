package gointerlock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

type GoInterval struct {
	Name           string
	Arg            func()
	Interval       time.Duration
	timer          *time.Timer
	RedisConnector *redis.Client
}

func (t *GoInterval) Run(ctx context.Context) error {

	if ctx == nil {
		ctx = context.Background()
	}

	if t.Interval == 0 {
		return errors.New("`Time Interval is missing!`")
	}
	if t.Arg == nil {
		return errors.New("`What this timer should to run?`")
	}

	var locker Locker
	locker.redisConnector = t.RedisConnector

	if locker.redisConnector == nil && t.Name == "" {
		return errors.New("`Distributed Jobs should have a unique name!`")
	}

	t.updateTimer()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Job %s is terminated", t.Name)
			return nil
		default:

			<-t.timer.C
			//lock
			locked, errLock := locker.Lock(ctx, t.Name)
			if errLock != nil {
				return errLock
			}
			if locked {
				// run the task
				t.Arg()
				//unlock
				errUnlock := locker.Unlock(ctx, t.Name)
				if errUnlock != nil {
					return errUnlock
				}
			}
			t.updateTimer()
		}
	}
}

func (t *GoInterval) updateTimer() {
	nextTick := time.Now()
	if !nextTick.After(time.Now()) {
		nextTick = nextTick.Add(t.Interval)
	}
	diff := nextTick.Sub(time.Now())
	if t.timer == nil {
		t.timer = time.NewTimer(diff)
	} else {
		t.timer.Reset(diff)
	}
}
