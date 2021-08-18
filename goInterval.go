package gointerlock

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var locker Lock

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

	//leave empty to get from ~/.aws/credentials, (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbRegion string

	//leave empty to get from ~/.aws/credentials
	AwsDynamoDbEndpoint string

	//leave empty to get from ~/.aws/credentials, StaticCredentials (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbAccessKeyID string

	//leave empty to get from ~/.aws/credentials, StaticCredentials (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbSecretAccessKey string

	//leave empty to get from ~/.aws/credentials, StaticCredentials (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbSessionToken string

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

	switch t.LockVendor {
	case RedisLock:
		r := &RedisLocker{
			redisConnector: t.RedisConnector,
			Name:           t.Name,
			RedisHost:      t.RedisHost,
			RedisPassword:  t.RedisPassword,
			RedisDB:        t.RedisDB,
		}
		err := r.SetClient()
		if err != nil {
			return err
		}

		locker = r
	case AwsDynamoDbLock:
		d := &DynamoDbLocker{
			AwsDynamoDbRegion:          t.AwsDynamoDbRegion,
			AwsDynamoDbEndpoint:        t.AwsDynamoDbEndpoint,
			AwsDynamoDbAccessKeyID:     t.AwsDynamoDbAccessKeyID,
			AwsDynamoDbSecretAccessKey: t.AwsDynamoDbSecretAccessKey,
			AwsDynamoDbSessionToken:    t.AwsDynamoDbSessionToken,
		}
		err := d.SetClient()
		if err != nil {
			return err
		}

		locker = d
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
			lock, err := t.isNotLockThenLock(ctx)
			if err != nil {
				log.Fatalf("err: %v", err)
				return nil
			}
			if lock {
				// run the task
				t.Arg()

				t.UnLock(ctx)
			}
			t.updateTimer()
		}
	}
}

func (t *GoInterval) isNotLockThenLock(ctx context.Context) (bool, error) {
	//lock
	if t.LockVendor == SingleApp {
		return true, nil
	}
	locked, err := locker.Lock(ctx, t.Name, t.Interval)

	if err != nil {
		log.Fatalf("err:%v", err)
		return false, err
	}
	return locked, nil
}

func (t *GoInterval) UnLock(ctx context.Context) {
	//unlock

	if t.LockVendor == SingleApp {
		return
	}

	err := locker.UnLock(ctx, t.Name)
	if err != nil {
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
