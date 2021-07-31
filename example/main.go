package main

import (
	"context"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	"log"
	"time"
)

func myJob() {
	fmt.Println(time.Now(), " - called")
}

func main() {
	cnx := context.Background()

	//test cron
	go func() {
		var job = gointerlock.GoInterval{
			Name:          "MyTestJob",
			Interval:      2 * time.Second,
			Arg:           myJob,
			RedisHost:     "localhost:6379",
			RedisPassword: "MyRedisPassword",
		}
		err := job.Run(cnx)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}()

	time.Sleep(10 * time.Second)
}
