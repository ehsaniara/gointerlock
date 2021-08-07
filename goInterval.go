package gointerlock

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/go-redis/redis/v8"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var locker Locker

type LockVendor int32

const (
	SingleApp       LockVendor = 0 // no distributed lock
	RedisLock       LockVendor = 1
	AwsDynamoDbLock LockVendor = 2
)

type GoInterval struct {

	//Name: is a unique job/task name, this is needed for distribution lock, this value enables the distribution mode. for local uses you don't need to set this value
	Name string

	// Arg: the func that need to be call in every period
	Arg func()

	// Interval: Timer Interval
	Interval time.Duration

	LockVendor LockVendor

	//redis connection---------

	// RedisConnector : in case your app has redis connection configured already
	RedisConnector *redis.Client

	// RedisHost Redis Host the default value "localhost:6379"
	RedisHost string

	// RedisPassword: Redis Password (AUTH), It can be blank if Redis has no authentication req
	RedisPassword string

	// 0 , It's from 0 to 15 (Not for redis cluster)
	RedisDB string

	// DynamoDb

	//leave empty to get from ~/.aws/credentials, (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbRegion string

	//leave empty to get from ~/.aws/credentials
	AwsDynamoDbEndpoint string

	//leave empty to get from ~/.aws/credentials, StaticCredentials (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbAccessKeyID string

	//leave empty to get from ~/.aws/credentials, StaticCredentials (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbSecretAccessKey string

	//leave empty to get from ~/.aws/credentials, StaticCredentials (if AwsDynamoDbEndpoint not provided)
	AwsDynamoDbSessionToken string

	// internal use, it should not get modified
	timer *time.Timer
}

// Run to start the interval timer
func (t *GoInterval) Run(ctx context.Context) error {

	if ctx == nil {
		ctx = context.Background()
	}

	if t.Interval == 0 {
		return errors.New("`Time Interval is missing!`")
	}

	if t.Arg == nil {
		return errors.New("`What this timer should to run?`")
	}

	//To check if it's a distributed system, support older version v1.0.3
	if t.Name != "" {
		if t.LockVendor == 0 {
			//default one, to support pre. Versions
			t.LockVendor = RedisLock
		}
	}

	err := t.init(ctx)
	if err != nil {
		return err
	}

	t.updateTimer()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Job %s terminated!", t.Name)
			return nil
		default:

			<-t.timer.C

			//lock
			lock, err := t.isNotLockThenLock(ctx)
			if err != nil {
				log.Fatalf("err: %v", err)
				return nil
			}
			if lock {
				// run the task
				t.Arg()

				t.UnLock(ctx)
			}
			t.updateTimer()
		}
	}
}

func (t *GoInterval) init(ctx context.Context) error {
	// distributed mod is enabled
	switch t.LockVendor {
	case RedisLock:

		//if given connection is null the use the built-in one
		if t.RedisConnector == nil {

			log.Printf("Job %s started in distributed mode!", t.Name)

			//if Redis host missed, use the default one
			if t.RedisHost == "" {
				t.RedisHost = "localhost:6379"
			}

			locker.redisConnector = redis.NewClient(&redis.Options{
				Addr:     t.RedisHost,
				Password: t.RedisPassword, // no password set
				DB:       0,               // use default DB
			})

		} else {
			// set the connection
			locker.redisConnector = t.RedisConnector
		}

		//validate the connection
		if locker.redisConnector.Conn(ctx) == nil {
			return errors.New("`Redis Connection Failed!`")
		}

		log.Printf("Job %s started in distributed mode by provided redis connection", t.Name)

	case AwsDynamoDbLock:

		// override the AWS profile credentials
		if aws.String(t.AwsDynamoDbEndpoint) == nil {
			// Initialize a session that the SDK will use to load
			// credentials from the shared credentials file ~/.aws/credentials
			// and region from the shared configuration file ~/.aws/config.
			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))
			// Create DynamoDB client
			locker.dynamoClient = dynamodb.New(sess)
		} else {

			if aws.String(t.AwsDynamoDbRegion) == nil {
				return errors.New("`AwsDynamoDbRegion is missing (AWS Region)`")
			}

			//setting StaticCredentials
			awsConfig := &aws.Config{
				Credentials: credentials.NewStaticCredentials(t.AwsDynamoDbAccessKeyID, t.AwsDynamoDbSecretAccessKey, t.AwsDynamoDbSessionToken),
				Region:      aws.String(t.AwsDynamoDbRegion),
				Endpoint:    aws.String(t.AwsDynamoDbEndpoint),
			}
			sess, err := session.NewSession(awsConfig)
			if err != nil {
				return err
			}
			// Create DynamoDB client
			locker.dynamoClient = dynamodb.New(sess)
		}

		//sess, err := session.NewSession(&aws.Config{
		//	Region:      aws.String("us-west-2"),
		//	Credentials: credentials.NewStaticCredentials(conf.AWS_ACCESS_KEY_ID, conf.AWS_SECRET_ACCESS_KEY, ""),
		//})

		if locker.dynamoClient == nil {
			return errors.New("`DynamoDb Connection Failed!`")
		}

		//check if table exist, if not create one
		tableInput := &dynamodb.CreateTableInput{
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("id"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("id"),
					KeyType:       aws.String("HASH"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(10),
				WriteCapacityUnits: aws.Int64(10),
			},
			//TimeToLiveDescription: &dynamodb.TimeToLiveDescription{
			//	AttributeName:    aws.String("ttl"),
			//	TimeToLiveStatus: aws.String("enable"),
			//},
			TableName: aws.String(Prefix),
		}

		_, err := locker.dynamoClient.CreateTable(tableInput)
		if err != nil {
			log.Printf("Got error calling CreateTable: %s", err)
		} else {
			fmt.Println("Created the table", Prefix)
		}

	default:

	}
	return nil
}

func (t *GoInterval) isNotLockThenLock(ctx context.Context) (bool, error) {

	// distributed mod is enabled
	switch t.LockVendor {
	case RedisLock:

		locked, err := locker.RedisLock(ctx, t.Name, t.Interval)

		if err != nil {
			return false, err
		}
		return locked, nil

	case AwsDynamoDbLock:

		locked, err := locker.DynamoDbLock(ctx, t.Name, t.Interval)

		if err != nil {
			return false, err
		}
		return locked, nil

	default:

		// no distributed lock
		return true, nil

	}
}

func (t *GoInterval) UnLock(ctx context.Context) {
	//unlock
	switch t.LockVendor {
	case RedisLock:

		err := locker.RedisUnlock(ctx, t.Name)
		if err != nil {
			return
		}

	case AwsDynamoDbLock:

		err := locker.DynamoDbUnlock(ctx, t.Name)
		if err != nil {
			return
		}

	default:

		// no distributed lock
		return
	}
}

func (t *GoInterval) updateTimer() {
	next := time.Now()
	if !next.After(time.Now()) {
		next = next.Add(t.Interval)
	}
	diff := next.Sub(time.Now())
	if t.timer == nil {
		t.timer = time.NewTimer(diff)
	} else {
		t.timer.Reset(diff)
	}
}
