package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/guilhermeais/event-processor/internal/infra/persister"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
	"github.com/guilhermeais/event-processor/internal/observability"
	"github.com/guilhermeais/event-processor/internal/usecases"
)

var v *validator.JSONSchemaValidator
var sqsClient *sqs.Client
var dlqURL string
var eventsTable string
var dynamoClient *dynamodb.Client

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
	sqsClient = sqs.NewFromConfig(cfg)

	schemaTable := os.Getenv("SCHEMA_TABLE_NAME")
	eventsTable = os.Getenv("EVENTS_TABLE_NAME")
	dlqURL = os.Getenv("DLQ_URL")

	l := validator.NewDynamoDbJSONSchemaLoader(dynamoClient, schemaTable)
	schemas, err := l.Load(ctx)
	if err != nil {
		log.Fatalf("error on loading schemas: %v", err)
	}
	v, err = validator.NewJSONSchemaValidator(schemas)
	if err != nil {
		log.Fatalf("error on creating validator: %v", err)
	}
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var batchItemFailures []events.SQSBatchItemFailure
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, message := range sqsEvent.Records {
		wg.Add(1)
		go func(msg events.SQSMessage) {
			slogJsonHandler := slog.NewJSONHandler(os.Stdout, nil)
			baseLogger := slog.New(slogJsonHandler)
			logger := observability.NewLogger(baseLogger)
			defer func() {
				logger.Emit("message processed")
			}()
			p := persister.NewDynamoPersister(dynamoClient, eventsTable, logger)
			processor := usecases.NewProcessor(v, p, logger)
			logger.AddAttribute("message_id", msg.MessageId)
			var cmd usecases.HandleCommand
			if attr, ok := msg.MessageAttributes["client_id"]; ok {
				cmd.ClientId = *attr.StringValue
				logger.AddAttribute("client_id", cmd.ClientId)
			}
			if attr, ok := msg.MessageAttributes["event_id"]; ok {
				cmd.EventId = *attr.StringValue
				logger.AddAttribute("event_id", cmd.EventId)
			}
			if attr, ok := msg.MessageAttributes["event_type"]; ok {
				cmd.EventType = *attr.StringValue
				logger.AddAttribute("event_type", cmd.EventType)
			}
			if err := json.Unmarshal([]byte(message.Body), &cmd.Payload); err != nil {
				logger.AddError(err)
				mu.Lock()
				batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
					ItemIdentifier: message.MessageId,
				})
				mu.Unlock()
				return
			}
			decision, _ := processor.Handle(ctx, cmd)
			if decision == usecases.DecisionAck {
				return
			}

			if decision == usecases.DecisionRetry {
				mu.Lock()
				batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
					ItemIdentifier: message.MessageId,
				})
				mu.Unlock()
				return
			}

			if decision == usecases.DecisionToDLQ {
				_, err := sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
					QueueUrl:    aws.String(dlqURL),
					MessageBody: aws.String(message.Body),
				})

				if err != nil {
					mu.Lock()
					batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
						ItemIdentifier: message.MessageId,
					})
					mu.Unlock()
				}
			}
		}(message)
	}

	wg.Wait()

	return events.SQSEventResponse{
		BatchItemFailures: batchItemFailures,
	}, nil
}

func main() {
	lambda.Start(handler)
}
