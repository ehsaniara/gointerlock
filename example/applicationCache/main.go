package main

import (
	"context"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	"log"
	"net/http"
	"time"
)

func main() {
	cnx := context.Background()

	var expensiveQueryResult = fmt.Sprintf("Last result at: %v", time.Now())

	//introduced the task
	go func() {
		//setting up the scheduler parameter
		var job = gointerlock.GoInterval{
			Interval: 5 * time.Second,
			Arg: func() {
				expensiveQueryResult = fmt.Sprintf("Last result at: %v", time.Now())
			},
		}

		// start the scheduler
		_ = job.Run(cnx)
	}()

	// http Server, access http://localhost:8080 from your browser
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprintln(writer, expensiveQueryResult)
	})
	fmt.Println("Server started at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
