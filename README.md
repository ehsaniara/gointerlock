# gointerlock

go interval job timer, with distributed lock

Quick Start

```shell
go get github.com/ehsaniara/gointerlock
```

## Simple Local Scheduler

(Interval every 2 seconds)

```go
var job = gointerlock.GoInterval{
    Interval: 2 * time.Second,
    Arg:      myJob,
}
err := jobTicker.Run(ctx)
if err != nil {
        log.Fatalf("Error: %s", err)
}
```

## Simple Distributed Scheduler

(Interval every 2 seconds)
you should already configure your Redis connection and pass it into the `GoInterLock`. Also make sure you are giving the
unique name per job

```go
var job = gointerlock.GoInterval{
    Interval: 2 * time.Second,
    Arg:      myJob,
    Name:     "MyTestJob",
    RedisConnector: pgeRedis.Rdb,
}
err := jobTicker.Run(ctx)
if err != nil {
    log.Fatalf("Error: %s", err)
}
```

in both examples `myJob` is your function, for example:

```go
func myJob() {
	fmt.Println(time.Now(), " - called")
}
```
_Note: currently `GoInterLock` does not support any argument for the job function_