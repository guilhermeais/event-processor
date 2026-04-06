package testhelpers

import (
	"context"
	"fmt"
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
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash, // Partition Key
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
	// Chave de partição composta por client_id e event_id, concatenados com "#"
	// Exemplo: "client123#event456"
	// Essa chave é usada para garantir a unicidade do evento e distribuir os dados de forma eficiente na tabela do DynamoDB.
	ClientEventID string `dynamodbav:"id"`
	ClientID      string `dynamodbav:"client_id"`
	EventID       string `dynamodbav:"event_id"`
	EventType     string `dynamodbav:"event_type"`
	Payload       string `dynamodbav:"payload"`
	CreatedAt     string `dynamodbav:"created_at"`
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
		"id": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", clientId, eventId)},
	}

	var parsedEvent EventDynamodb
	saved, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key:       firstClientIdKey,
	})
	assert.NoError(t, err)
	err = attributevalue.UnmarshalMap(saved.Item, &parsedEvent)
	assert.NoError(t, err)

	return parsedEvent
}
