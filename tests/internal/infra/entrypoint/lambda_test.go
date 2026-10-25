package entrypoint

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/guilhermeais/event-processor/internal/infra/entrypoint"
	testhelpers "github.com/guilhermeais/event-processor/tests/testhelpers"
	"github.com/stretchr/testify/assert"
)

type sut struct {
	sut                           *entrypoint.LambdaEntryPoint
	eventsTbl, schemasTbl, dlqUrl string
	cleanup                       func()
}

func makeSut(t *testing.T, ctx context.Context, dynamo *dynamodb.Client, sqsClientLocal *sqs.Client) sut {
	testingDeps := testhelpers.PrepareDependencies(ctx, t, dynamo, sqsClientLocal)

	l := entrypoint.NewLambdaEntryPoint(
		dynamo,
		testingDeps.EventsTbl,
		testingDeps.DLQUrl,
		testingDeps.Validator,
		sqsClientLocal,
		testhelpers.TestLoggerFactory(t),
	)

	return sut{
		sut:        l,
		eventsTbl:  testingDeps.EventsTbl,
		schemasTbl: testingDeps.SchemasTbl,
		dlqUrl:     testingDeps.DLQUrl,
		cleanup:    testingDeps.Cleanup,
	}
}

func TestHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, cleanup := testhelpers.SetupLocalStack(ctx)
	defer cleanup()

	dynamo := dynamodb.NewFromConfig(cfg)
	sqsClientLocal := sqs.NewFromConfig(cfg)

	t.Run("should ack if everthing is OK", func(t *testing.T) {
		s := makeSut(t, ctx, dynamo, sqsClientLocal)
		defer s.cleanup()

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

		resp, err := s.sut.Handler(ctx, ev)

		itemOnDb := testhelpers.GetDynamoDbEvent(t, ctx, dynamo, s.eventsTbl, "client-1", "evt-1")
		assert.NotEmpty(t, itemOnDb)
		assert.Equal(t, itemOnDb.ClientID, "client-1")
		assert.Equal(t, itemOnDb.EventID, "evt-1")
		assert.Equal(t, itemOnDb.EventType, "profile.created")
		assert.Equal(t, itemOnDb.Payload, payload)

		assert.NoError(t, err)
		assert.Len(t, resp.BatchItemFailures, 0)
	})

	t.Run("should move to dlq", func(t *testing.T) {
		s := makeSut(t, ctx, dynamo, sqsClientLocal)
		defer s.cleanup()

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

		resp, err := s.sut.Handler(ctx, ev)

		assert.NoError(t, err)
		assert.Len(t, resp.BatchItemFailures, 0)

		itemOnDb := testhelpers.GetDynamoDbEvent(t, ctx, dynamo, s.eventsTbl, "client-1", "evt-2")
		assert.Empty(t, itemOnDb)
		dlqMessages, err := sqsClientLocal.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(s.dlqUrl),
			MessageAttributeNames: []string{"All"},
		})
		assert.Len(t, dlqMessages.Messages, 1, "Esperado exatamente 1 mensagem na DLQ")

		msg := dlqMessages.Messages[0]
		assert.Equal(t, payload, *msg.Body, "O corpo da mensagem na DLQ deve corresponder ao payload original")

		assert.Equal(t, "client-1", *msg.MessageAttributes["client_id"].StringValue)
		assert.Equal(t, "evt-2", *msg.MessageAttributes["event_id"].StringValue)
		assert.Equal(t, "profile.created", *msg.MessageAttributes["event_type"].StringValue)
	})
}
