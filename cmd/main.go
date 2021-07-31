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

	//test cron
	var job = gointerlock.GoInterval{
		Name:     "MyTestJob",
		Interval: 2 * time.Second,
		Arg:      myJob,
	}
	err := job.Run(context.Background())
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

}
