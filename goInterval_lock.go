package gointerlock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/go-redis/redis/v8"
)

const Prefix = "GoInterLock"

type Lock interface {
	Lock(ctx context.Context, key string, interval time.Duration) (success bool, err error)
	UnLock(ctx context.Context, key string) (err error)
	SetClient() error
}

type RedisLocker struct {
	redisConnector *redis.Client

	// Name: is a unique job/task name, this is needed for distribution lock, this value enables the distribution mode. for local uses you don't need to set this value
	Name string

	// RedisHost Redis Host the default value "localhost:6379"
	RedisHost string

	// RedisPassword: Redis Password (AUTH), It can be blank if Redis has no authentication req
	RedisPassword string

	// 0 , It's from 0 to 15 (Not for redis cluster)
	RedisDB string
}

func (r *RedisLocker) SetClient() error {

	//already has connection
	if r.redisConnector != nil {
		return nil
	}

	log.Printf("Job %s started in distributed mode!", r.Name)

	//if Redis host missed, use the default one
	if r.RedisHost == "" {
		r.RedisHost = "localhost:6379"
	}

	r.redisConnector = redis.NewClient(&redis.Options{
		Addr:     r.RedisHost,
		Password: r.RedisPassword, // no password set
		DB:       0,               // use default DB
	})

	log.Printf("Job %s started in distributed mode by provided redis connection", r.Name)
	return nil
}

func (r *RedisLocker) Lock(ctx context.Context, key string, lockTtl time.Duration) (success bool, err error) {

	if r.redisConnector != nil {

		if key == "" {
			return false, errors.New("`Distributed Jobs should have a unique name!`")
		}

		res, err := r.redisConnector.SetNX(ctx, fmt.Sprintf("%s_%s", Prefix, key), time.Now().String(), lockTtl).Result()
		if err != nil {
			return false, err
		}
		return res, nil
	}

	return false, errors.New("`No Redis Connection found`")
}

func (r *RedisLocker) UnLock(ctx context.Context, key string) error {
	if r.redisConnector != nil {
		return r.redisConnector.Del(ctx, fmt.Sprintf("%s_%s", Prefix, key)).Err()
	} else {
		return nil
	}
}

type DynamoDbLocker struct {
	dynamoClient *dynamodb.DynamoDB

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
}

func (d *DynamoDbLocker) SetClient() error {

	// override the AWS profile credentials
	if aws.String(d.AwsDynamoDbEndpoint) == nil {
		// Initialize a session that the SDK will use to load
		// credentials from the shared credentials file ~/.aws/credentials
		// and region from the shared configuration file ~/.aws/config.
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		// Create DynamoDB client
		d.dynamoClient = dynamodb.New(sess)
	} else {

		if aws.String(d.AwsDynamoDbRegion) == nil {
			return errors.New("`AwsDynamoDbRegion is missing (AWS Region)`")
		}

		//setting StaticCredentials
		awsConfig := &aws.Config{
			Credentials: credentials.NewStaticCredentials(d.AwsDynamoDbAccessKeyID, d.AwsDynamoDbSecretAccessKey, d.AwsDynamoDbSessionToken),
			Region:      aws.String(d.AwsDynamoDbRegion),
			Endpoint:    aws.String(d.AwsDynamoDbEndpoint),
		}
		sess, err := session.NewSession(awsConfig)
		if err != nil {
			return err
		}
		// Create DynamoDB client
		d.dynamoClient = dynamodb.New(sess)
	}

	//sess, err := session.NewSession(&aws.Config{
	//	Region:      aws.String("us-west-2"),
	//	Credentials: credentials.NewStaticCredentials(conf.AWS_ACCESS_KEY_ID, conf.AWS_SECRET_ACCESS_KEY, ""),
	//})

	if d.dynamoClient == nil {
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

	_, err := d.dynamoClient.CreateTable(tableInput)
	if err != nil {
		log.Printf("Got error calling CreateTable: %s", err)
	} else {
		fmt.Println("Created the table", Prefix)
	}

	return nil
}

func (d *DynamoDbLocker) Lock(ctx context.Context, key string, lockTtl time.Duration) (success bool, err error) {

	if d.dynamoClient == nil {
		return false, errors.New("`No Redis Connection found`")
	}

	expr, _ := expression.NewBuilder().WithFilter(
		//filter by id
		expression.Name("id").Equal(expression.Value(key)),
	).Build()

	// Build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		TableName:                 aws.String(Prefix),
	}

	// Make the DynamoDB Query API call
	result, _ := d.dynamoClient.ScanWithContext(ctx, params)

	if len(result.Items) > 0 {
		return false, nil
	}

	_, errPut := d.dynamoClient.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		Item:      DynamoDbUnlockMarshal(key),
		TableName: aws.String(Prefix),
	})

	if errPut != nil {
		return false, errPut
	}

	return true, nil

}

func (d *DynamoDbLocker) UnLock(ctx context.Context, key string) error {
	if d.dynamoClient != nil {
		_, _ = d.dynamoClient.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
			Key:       DynamoDbUnlockMarshal(key),
			TableName: aws.String(Prefix),
		})
	}
	return nil
}

func DynamoDbUnlockMarshal(key string) map[string]*dynamodb.AttributeValue {
	lockObj, _ := dynamodbattribute.MarshalMap(struct {
		Id string `json:"id"`
	}{
		Id: key,
	})
	return lockObj
}
