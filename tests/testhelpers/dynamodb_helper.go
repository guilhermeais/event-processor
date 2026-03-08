package testhelpers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

func CreateEventsTable(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("client_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("event_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("client_id"),
				KeyType:       types.KeyTypeHash, // Partition Key
			},
			{
				AttributeName: aws.String("event_id"),
				KeyType:       types.KeyTypeRange, // Sort Key
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	})
	return err
}

func CreateSchemasTable(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("event_type"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("event_type"),
				KeyType:       types.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	})
	return err
}

func DeleteTable(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: &tableName,
	})
	return err
}

type EventDynamodb struct {
	ClientID  string `dynamodbav:"client_id"`
	EventID   string `dynamodbav:"event_id"`
	EventType string `dynamodbav:"event_type"`
	Payload   string `dynamodbav:"payload"`
	CreatedAt string `dynamodbav:"created_at"`
}

func GetDynamoDbEvent(
	t *testing.T,
	ctx context.Context,
	dynamoClient *dynamodb.Client,
	tableName,
	clientId,
	eventId string,
) EventDynamodb {
	firstClientIdKey := map[string]types.AttributeValue{
		"client_id": &types.AttributeValueMemberS{Value: clientId},
		"event_id":  &types.AttributeValueMemberS{Value: eventId},
	}

	var parsedEvent EventDynamodb
	saved, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key:       firstClientIdKey,
	})
	err = attributevalue.UnmarshalMap(saved.Item, &parsedEvent)
	assert.Nil(t, err)

	return parsedEvent
}
