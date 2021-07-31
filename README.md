# gointerlock

go interval job timer, with distributed lock

Quick Start

```shell
go get github.com/ehsaniara/gointerlock
```

## Simple Local Scheduler (Single App)

(Interval every 2 seconds)

```go
var job = gointerlock.GoInterval{
    Interval: 2 * time.Second,
    Arg:      myJob,
}
err := job.Run(ctx)
if err != nil {
        log.Fatalf("Error: %s", err)
}
```

## Simple Distributed Scheduler (Scaled)

(Interval every 2 seconds)
you should already configure your Redis connection and pass it into the `GoInterLock`. Also make sure you are giving the
unique name per job

Step 1: configure redis connection `redisConnection.Rdb` from the existing application and pass it into the Job. for example:
```go
var redisConnector = redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "myRedisPassword", 
    DB:       0,               
})
```
Step 2: Pass the redis connection into the `GoInterval`

```go
var job = gointerlock.GoInterval{
    Interval: 2 * time.Second,
    Arg:      myJob,
    Name:     "MyTestJob",
    RedisConnector: redisConnector,
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

### Built in Redis Config

another way is to use an existing redis connection:

```go
var job = gointerlock.GoInterval{
    Name:          "MyTestJob",
    Interval:      2 * time.Second,
    Arg:           myJob,
    RedisHost:     "localhost:6379",
    RedisPassword: "myRedisPassword",
}
err := job.Run(context.Background())
if err != nil {
    log.Fatalf("Error: %s", err)
}
```