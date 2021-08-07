package gointerlock

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

const Prefix = "GoInterLock"

type Locker struct {
	redisConnector *redis.Client
	dynamoClient   *dynamodb.DynamoDB
}

func (s *Locker) RedisLock(ctx context.Context, key string, lockTtl time.Duration) (success bool, err error) {

	if s.redisConnector != nil {

		if key == "" {
			return false, errors.New("`Distributed Jobs should have a unique name!`")
		}

		res, err := s.redisConnector.SetNX(ctx, fmt.Sprintf("%s_%s", Prefix, key), time.Now().String(), lockTtl).Result()
		if err != nil {
			return false, err
		}
		return res, nil
	}

	return false, errors.New("`No Redis Connection found`")
}

func (s *Locker) RedisUnlock(ctx context.Context, key string) error {
	if s.redisConnector != nil {
		return s.redisConnector.Del(ctx, fmt.Sprintf("%s_%s", Prefix, key)).Err()
	} else {
		return nil
	}
}

func (s *Locker) DynamoDbLock(ctx context.Context, key string, lockTtl time.Duration) (success bool, err error) {

	if s.dynamoClient == nil {
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
	result, _ := s.dynamoClient.ScanWithContext(ctx, params)

	log.Printf("--------------result.Items: %s", result.Items)
	if len(result.Items) > 0 {
		return false, nil
	}

	insert, errPut := s.dynamoClient.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		Item:      DynamoDbUnlockMarshal(key),
		TableName: aws.String(Prefix),
	})

	if errPut != nil {
		return false, errPut
	}

	log.Printf("--------------insert: %s", insert)

	return true, nil

}

func (s *Locker) DynamoDbUnlock(ctx context.Context, key string) error {
	if s.dynamoClient != nil {
		item, _ := s.dynamoClient.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
			Key:       DynamoDbUnlockMarshal(key),
			TableName: aws.String(Prefix),
		})
		log.Printf("DynamoDbUnlock:%s\n", item)
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
