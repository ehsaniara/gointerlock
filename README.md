# goInterLock 
![Go Interval Lock](material/gointerlock_bg.png)

_known as: ⏰ Interval (Cron / Job / Task / Scheduler) Go Centralized Lock ⏱️_

## Go Interval job/task , with centralized Lock for Distributed Systems

`goInterLock` is go job/task scheduler with centralized locking mechanism. In distributed system locking is preventing task been executed in every instant that has the scheduler, 

>  **_Note:_** Kubernetes has the Lease object, which you can use to implement the `Leader election`. [Here is a working example of using kubernetes Lease API](K8sLease.md)  

**For example:** if your application has a task of calling some external APIs or doing some DB querying every 10 minutes, the lock prevents the process been run in every instance of that application, and you ended up running that task multiple time every 10 minutes.

Quick Start

```shell
go get github.com/ehsaniara/gointerlock
```

Supported Lock
- [Redis](#redis)
- [AWS DynamoDB](#aws-dynamodb)
- Postgres DB

# Local Scheduler (Single App)

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

### Examples
[**Basic Local Task:**](example/basicLocal/main.go) Simple Task Interval (Single App).

[**Application Cache:**](./example/applicationCache/main.go) An example of periodically cached value update on http server.

------
# Distributed Mode (Scaled Up)

## Redis (Recommended)

### Existing Redis Connection
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
> Currently `GoInterLock` does not support any argument for the job function

### Built in Redis Connector

another way is to use an existing redis connection:

```go
var job = gointerlock.GoInterval{
    Name:          "MyTestJob",
    Interval:      2 * time.Second,
    Arg:           myJob,
    RedisHost:     "localhost:6379",
    RedisPassword: "myRedisPassword", //if no pass leave it as ""
}
err := job.Run(context.Background())
if err != nil {
    log.Fatalf("Error: %s", err)
}
```

##### GoInterLock is using [go-redis](https://github.com/go-redis/redis) for Redis Connection.



### Examples

[**Basic Distributed Task:**](example/redis/basic/main.go) Simple Task Interval with Redis Lock.

-----

## AWS DynamoDb

### Basic Config (Local Environment)
This ia sample of local DynamoDb (Docker) for your local test. 
```go
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
```
task:
```go
func myJob() {
	fmt.Println(time.Now(), " - called")
}
```
> You can get the docker-compose file from [AWS DynamoDB Docker compose](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/DynamoDBLocal.DownloadingAndRunning.html) , also avalble from: [docker-compose.yml](./example/awsDynamoDb/docker-compose.yml).

### Using AWS Profile

`goInterLock` will get credentials from the AWS profile

```go
var job = gointerlock.GoInterval{
    Name:                       "MyTestJob",
    Interval:                   2 * time.Second,
    Arg:                        myJob,
}
err := job.Run(cnx)
if err != nil {
    log.Fatalf("Error: %s", err)
}
```
### Examples

[**Basic Distributed Task:**](example/awsDynamoDb/main.go) Simple Task Interval with DynamoDb Lock.



# :four_leaf_clover: Centralized lock for the Distributed Systems (No Timer/Scheduler)

```go

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type DistributedLock interface {
	Lock(ctx context.Context, lockName string, maxLockingDuration time.Duration) bool
	UnLock(ctx context.Context, lockName string)
}

func NewDistributedLock(rdb *redis.Client) DistributedLock {
	return &distributedLock{
		rdb: rdb,
	}
}

type distributedLock struct {
	rdb *redis.Client
}

// Lock return TRUE when successfully locked, return FALSE if it's already been locked by others
func (d distributedLock) Lock(ctx context.Context, lockName string, maxLockingDuration time.Duration) bool {
	key := fmt.Sprintf("lock_%s", lockName)
	//check if it's already locked
	iter := d.rdb.Scan(ctx, 0, key, 0).Iterator()
	for iter.Next(ctx) {
		//exit if lock exist
		return false
	}
	//then lock it then
	d.rdb.Set(ctx, key, []byte("true"), maxLockingDuration)
	return true
}

func (d distributedLock) UnLock(ctx context.Context, lockName string) {
	key := fmt.Sprintf("lock_%s", lockName)
	//remove the lock
	d.rdb.Del(ctx, key)
}

```
