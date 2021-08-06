package gointerlock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

var locker Locker

type LockVendor int32

const (
	SingleApp       LockVendor = 0 // no distributed lock
	RedisLock       LockVendor = 1
	AwsDynamoDbLock LockVendor = 2
)

type GoInterval struct {

	//Name: is a unique job/task name, this is needed for distribution lock, this value enables the distribution mode. for local uses you don't need to set this value
	Name string

	// Arg: the func that need to be call in every period
	Arg func()

	// Interval: Timer Interval
	Interval time.Duration

	LockVendor LockVendor

	//redis connection---------

	// RedisConnector : in case your app has redis connection configured already
	RedisConnector *redis.Client

	// RedisHost Redis Host the default value "localhost:6379"
	RedisHost string

	// RedisPassword: Redis Password (AUTH), It can be blank if Redis has no authentication req
	RedisPassword string

	// 0 , It's from 0 to 15 (Not for redis cluster)
	RedisDB string

	// DynamoDb

	// internal use, it should not get modified
	timer *time.Timer
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

	//To check if it's a distributed system, support older version v1.0.3
	if t.Name != "" {
		if t.LockVendor == 0 {
			//default one, to support pre. Versions
			t.LockVendor = RedisLock
		}
	}

	err := t.init(ctx)
	if err != nil {
		return err
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
			lock, _ := t.isNotLockThenLock(ctx)
			if lock {

				// run the task
				t.Arg()

				t.UnLock(ctx)
			}
			t.updateTimer()
		}
	}
}

func (t *GoInterval) init(ctx context.Context) error {
	// distributed mod is enabled
	switch t.LockVendor {
	case RedisLock:

		//if given connection is null the use the built-in one
		if t.RedisConnector == nil {

			log.Printf("Job %s started in distributed mode!", t.Name)

			//if Redis host missed, use the default one
			if t.RedisHost == "" {
				t.RedisHost = "localhost:6379"
			}

			locker.redisConnector = redis.NewClient(&redis.Options{
				Addr:     t.RedisHost,
				Password: t.RedisPassword, // no password set
				DB:       0,               // use default DB
			})

		} else {
			// set the connection
			locker.redisConnector = t.RedisConnector
		}

		//validate the connection
		if locker.redisConnector.Conn(ctx) == nil {
			return errors.New("`Redis Connection Failed!`")
		}

		log.Printf("Job %s started in distributed mode by provided redis connection", t.Name)

	case AwsDynamoDbLock:

	default:

	}
	return nil
}

func (t *GoInterval) isNotLockThenLock(ctx context.Context) (bool, error) {

	// distributed mod is enabled
	switch t.LockVendor {
	case RedisLock:

		locked, err := locker.RedisLock(ctx, t.Name, t.Interval)

		if err != nil {
			return false, err
		}
		return locked, nil

	case AwsDynamoDbLock:

		//TODO: Should be implemented
		return true, nil

	default:

		// no distributed lock
		return true, nil

	}
}

func (t *GoInterval) UnLock(ctx context.Context) {
	//unlock
	switch t.LockVendor {
	case RedisLock:

		err := locker.RedisUnlock(ctx, t.Name)
		if err != nil {
			return
		}

	case AwsDynamoDbLock:

		//TODO: Should be implemented
		return

	default:

		// no distributed lock
		return
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
