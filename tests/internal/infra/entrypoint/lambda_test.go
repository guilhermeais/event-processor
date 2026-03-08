package entrypoint

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/guilhermeais/event-processor/internal/infra/entrypoint"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
	testhelpers "github.com/guilhermeais/event-processor/tests/testhelpers"
	"github.com/stretchr/testify/assert"
)

func prepareTables(t *testing.T, ctx context.Context, dynamo *dynamodb.Client) (func(), string, string) {
	eventsTbl := fmt.Sprintf("events_%s", uuid.NewString())
	schemasTbl := fmt.Sprintf("schemas_%s", uuid.NewString())
	assert.NoError(t, testhelpers.CreateEventsTable(ctx, dynamo, eventsTbl))
	assert.NoError(t, testhelpers.CreateSchemasTable(ctx, dynamo, schemasTbl))

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
		defer testhelpers.DeleteTable(ctx, dynamo, eventsTbl)
		defer testhelpers.DeleteTable(ctx, dynamo, schemasTbl)
	}, eventsTbl, schemasTbl
}

func TestHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, cleanup := testhelpers.SetupLocalStack(ctx)
	defer cleanup()

	dynamo := dynamodb.NewFromConfig(cfg)
	sqsClientLocal := sqs.NewFromConfig(cfg)

	cleanupTables, eventsTbl, schemasTbl := prepareTables(t, ctx, dynamo)
	defer cleanupTables()

	dlqOut, err := sqsClientLocal.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("dlq-" + uuid.NewString()),
	})
	assert.NoError(t, err)

	loader := validator.NewDynamoDbJSONSchemaLoader(dynamo, schemasTbl)
	schemas, err := loader.Load(ctx)
	assert.NoError(t, err)

	v, err := validator.NewJSONSchemaValidator(schemas)
	assert.NoError(t, err)

	sut := entrypoint.NewLambdaEntryPoint(dynamo, eventsTbl, *dlqOut.QueueUrl, v, sqsClientLocal)

	t.Run("should ack if everthing is OK", func(t *testing.T) {
		payload := `{"id":"1","birthday":"2003-08-26"}`
		ev := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId: "m-1",
					Body:      payload,
					MessageAttributes: map[string]events.SQSMessageAttribute{
						"client_id":  {DataType: "String", StringValue: aws.String("client-1")},
						"event_id":   {DataType: "String", StringValue: aws.String("evt-1")},
						"event_type": {DataType: "String", StringValue: aws.String("profile.created")},
					},
				},
			},
		}

		resp, err := sut.Handler(ctx, ev)

		itemOnDb := testhelpers.GetDynamoDbEvent(t, ctx, dynamo, eventsTbl, "client-1", "evt-1")
		assert.NotEmpty(t, itemOnDb)
		assert.Equal(t, itemOnDb.ClientID, "client-1")
		assert.Equal(t, itemOnDb.EventID, "evt-1")
		assert.Equal(t, itemOnDb.EventType, "profile.created")
		assert.Equal(t, itemOnDb.Payload, payload)

		assert.NoError(t, err)
		assert.Len(t, resp.BatchItemFailures, 0)
	})

	t.Run("should move to dlq", func(t *testing.T) {
		payload := `{"id": {"id":"invalid"}}`
		ev := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId: "m-2",
					Body:      payload,
					MessageAttributes: map[string]events.SQSMessageAttribute{
						"client_id":  {DataType: "String", StringValue: aws.String("client-1")},
						"event_id":   {DataType: "String", StringValue: aws.String("evt-2")},
						"event_type": {DataType: "String", StringValue: aws.String("profile.created")},
					},
				},
			},
		}

		resp, err := sut.Handler(ctx, ev)

		itemOnDb := testhelpers.GetDynamoDbEvent(t, ctx, dynamo, eventsTbl, "client-1", "evt-2")
		assert.Empty(t, itemOnDb)
		assert.NoError(t, err)
		assert.Len(t, resp.BatchItemFailures, 1)
	})
}
