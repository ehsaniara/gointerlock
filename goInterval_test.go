package gointerlock_test

import (
	"context"
	"flag"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	"github.com/go-redis/redis/v8"
	"log"
	"testing"
	"time"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)
var (
	RedisHost string
	RedisPass string
)

func init() {
	flag.StringVar(&RedisPass, "RedisPass", "", "Redis Password")
	flag.StringVar(&RedisHost, "RedisHost", "localhost:6379", "Redis Host")
}

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

func TestGoInterval_Run(t *testing.T) {
	var counter = 0
	var job1 = gointerlock.GoInterval{
		Interval: 100 * time.Millisecond,
		Arg: func() {
			counter++
		},
	}
	go func() {
		_ = job1.Run(context.Background())
	}()
	time.Sleep(2 * time.Second)

	log.Printf("counter %d", counter)
	if counter != 19 {
		t.Error("counter should be 19")
	}

}

func TestGoInterval_LockCheck(t *testing.T) {
	var counter = 0
	cnx := context.Background()
	var job2 = gointerlock.GoInterval{
		Name:          "job2",
		RedisHost:     RedisHost,
		RedisPassword: RedisPass,
		Interval:      1 * time.Second,
		Arg: func() {
			counter++
		},
	}
	time.Sleep(1 * time.Second)
	//instance 1 replication
	go func() {
		_ = job2.Run(cnx)
	}()
	//instance 2 replication
	go func() {
		_ = job2.Run(cnx)
	}()
	//instance 3 replication
	go func() {
		_ = job2.Run(cnx)
	}()
	time.Sleep(2 * time.Second)

	log.Printf("counter %d", counter)
	if counter != 1 {
		t.Error("counter should be 1")
	}
}
