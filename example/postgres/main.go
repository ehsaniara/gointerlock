package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ehsaniara/gointerlock"
)

func myJob(s string) {
	fmt.Printf("%s - called %s\n", time.Now().String(), s)
}

func main() {
	cnx := context.Background()

	//test cron
	go func() {

		var job = gointerlock.GoInterval{
			Name:     "MyTestJob",
			Interval: 2 * time.Second,
			Arg: func() {
				myJob("job1")
			},
			LockVendor:      gointerlock.PostgresLock,
			PostgresConnStr: "postgresql://guest:guest@localhost:5432/locks?sslmode=disable",
		}
		err := job.Run(cnx)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}()

	//test cron
	go func() {

		var job = gointerlock.GoInterval{
			Name:     "MyTestJob",
			Interval: 3 * time.Second,
			Arg: func() {
				myJob("job2")
			},
			LockVendor:      gointerlock.PostgresLock,
			PostgresConnStr: "postgresql://guest:guest@localhost:5432/locks?sslmode=disable",
		}
		err := job.Run(cnx)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}()

	//example: just run it for 10 second before application exits
	time.Sleep(20 * time.Second)
}
