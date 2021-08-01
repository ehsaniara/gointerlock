package gointerlock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

type GoInterval struct {

	//Name: is a unique job/task name, this is needed for distribution lock, this value enables the distribution mode
	// for local uses you don't need to set this value
	Name string

	// Arg: the func that need to be call in every period
	Arg func()

	// Interval: Timer Interval
	Interval time.Duration

	// internal use, it should not get modified
	timer *time.Timer

	// RedisConnector : in case your app has redis connection configured already
	RedisConnector *redis.Client

	// RedisHost Redis Host the default value "localhost:6379"
	RedisHost string

	// RedisPassword: Redis Password (AUTH), It can be blank if Redis has no authentication req
	RedisPassword string

	// 0 , It's from 0 to 15
	RedisDB string
}

// Run to start the interval timer
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

	// distributed mod is enabled
	if t.RedisConnector != nil {

		//validate the connection
		if t.RedisConnector.Conn(ctx) == nil {
			return errors.New("`Invalid Redis Connection`")
		}
		locker.redisConnector = t.RedisConnector
		log.Printf("Job %s started in distributed mode by provided redis connection", t.Name)

	}

	// distributed mod is enabled when name is provided, By using GoInterLock's provided connector
	if t.RedisConnector == nil && t.Name != "" {

		log.Printf("Job %s started in distributed mode!", t.Name)

		//if Redis host missed
		if t.RedisHost == "" {
			t.RedisHost = "localhost:6379"
		}

		locker.redisConnector = redis.NewClient(&redis.Options{
			Addr:     t.RedisHost,
			Password: t.RedisPassword, // no password set
			DB:       0,               // use default DB
		})
	}

	t.updateTimer()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Job %s terminated!", t.Name)
			return nil
		default:

			<-t.timer.C

			//lock
			locked, errLock := locker.Lock(ctx, t.Name, t.Interval)

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
	next := time.Now()
	if !next.After(time.Now()) {
		next = next.Add(t.Interval)
	}
	diff := next.Sub(time.Now())
	if t.timer == nil {
		t.timer = time.NewTimer(diff)
	} else {
		t.timer.Reset(diff)
	}
}
