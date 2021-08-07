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
			Name:                       "MyTestJob",
			Interval:                   2 * time.Second,
			Arg:                        myJob,
			LockVendor:                 gointerlock.AwsDynamoDbLock,
			AwsDynamoDbRegion:          "us-east-1",
			AwsDynamoDbEndpoint:        "http://127.0.0.1:8000",
			AwsDynamoDbSecretAccessKey: "dummy",
			AwsDynamoDbAccessKeyID:     "dummy",
		}
		err := job.Run(cnx)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}()

	//example: just run it for 10 second before application exits
	time.Sleep(10 * time.Second)
}
