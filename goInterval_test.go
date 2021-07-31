package gointerlock_test

import (
	"context"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

const RedisHost string = "localhost:6379"
const RedisPass string = "password"

var (
	ctx = context.Background()
	rdb *redis.Client
)

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:         ":6379",
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})
}

func ExampleNewClient() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     RedisHost, // use default Addr
		Password: RedisPass, // no password set
		DB:       0,         // use default DB
	})

	pong, err := rdb.Ping(ctx).Result()
	fmt.Println(pong, err)
	// Output: PONG <nil>
}

func ExampleGoInterval_Run() {
	go func() {
		var job = gointerlock.GoInterval{
			Interval: 1 * time.Second,
			Arg: func() {
				fmt.Print("called-")
			},
		}
		err := job.Run(context.Background())
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}()
	time.Sleep(4 * time.Second)
	// Output: called-called-called-
}
