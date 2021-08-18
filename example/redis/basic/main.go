package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ehsaniara/gointerlock"
)

func myJob() {
	fmt.Println(time.Now(), " - called")
}

func main() {
	cnx := context.Background()

	var job = gointerlock.GoInterval{
		LockVendor:    gointerlock.RedisLock,
		Name:          "MyTestJob",
		Interval:      2 * time.Second,
		Arg:           myJob,
		RedisHost:     "localhost:6379",
		RedisPassword: "secret",
	}

	//test cron
	go func() {
		err := job.Run(cnx)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}()

	//example: just run it for 10 second before application exits
	time.Sleep(10 * time.Second)
}
