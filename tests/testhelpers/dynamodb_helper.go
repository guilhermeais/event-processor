package testhelpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
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

func PrepareDependencies(ctx context.Context, t *testing.T, dynamo *dynamodb.Client, sqsClientLocal *sqs.Client) TestingDependencies {
	cleanupTables, eventsTbl, schemasTbl := PrepareTables(t, ctx, dynamo)

	dlqOut, err := sqsClientLocal.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("dlq-" + uuid.NewString()),
	})
	assert.NoError(t, err)

	loader := validator.NewDynamoDbJSONSchemaLoader(dynamo, schemasTbl)
	schemas, err := loader.Load(ctx)
	assert.NoError(t, err)

	v, err := validator.NewJSONSchemaValidator(schemas)
	assert.NoError(t, err)
	return TestingDependencies{
		EventsTbl:  eventsTbl,
		SchemasTbl: schemasTbl,
		DLQUrl:     *dlqOut.QueueUrl,
		Validator:  v,
		Cleanup: func() {
			cleanupTables()
		},
	}
}

type TestingDependencies struct {
	EventsTbl, SchemasTbl, DLQUrl string
	Validator                     *validator.JSONSchemaValidator
	Cleanup                       func()
}

func PrepareTables(t *testing.T, ctx context.Context, dynamo *dynamodb.Client) (func(), string, string) {
	eventsTbl := fmt.Sprintf("events_%s", uuid.NewString())
	schemasTbl := fmt.Sprintf("schemas_%s", uuid.NewString())
	assert.NoError(t, CreateEventsTable(ctx, dynamo, eventsTbl))
	assert.NoError(t, CreateSchemasTable(ctx, dynamo, schemasTbl))

	_, err := dynamo.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(schemasTbl),
		Item: map[string]ddbtypes.AttributeValue{
			"event_type": &ddbtypes.AttributeValueMemberS{Value: "profile.created"},
			"schema": &ddbtypes.AttributeValueMemberS{Value: `{
				"type":"object",
				"properties": {
					"id": { "type": "string" },
					"birthday": { "type": "string", "format": "date" }
				}
			}`},
		},
	})
	assert.NoError(t, err)

	return func() {
		defer DeleteTable(ctx, dynamo, eventsTbl)
		defer DeleteTable(ctx, dynamo, schemasTbl)
	}, eventsTbl, schemasTbl
}
