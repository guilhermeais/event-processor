package entrypoint

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/guilhermeais/event-processor/internal/infra/persister"
	"github.com/guilhermeais/event-processor/internal/observability"
	"github.com/guilhermeais/event-processor/internal/ports"
	"github.com/guilhermeais/event-processor/internal/usecases"
)

type LambdaEntryPoint struct {
	dynamoClient    *dynamodb.Client
	eventsTableName string
	dlqURL          string
	validator       ports.Validator
	sqsClient       *sqs.Client
}

func getStringMessageAttribute(msg events.SQSMessage, key string) string {
	attr, ok := msg.MessageAttributes[key]
	if !ok || attr.StringValue == nil {
		return ""
	}

	return *attr.StringValue
}

func (l *LambdaEntryPoint) Handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var batchItemFailures []events.SQSBatchItemFailure
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, message := range sqsEvent.Records {
		wg.Add(1)
		go func(msg events.SQSMessage) {
			logger := observability.NewLoggerDefault()
			defer func() {
				logger.Emit("message processed")
				wg.Done()
			}()
			logger.AddAttribute("message_id", msg.MessageId)
			p := persister.NewDynamoPersister(l.dynamoClient, l.eventsTableName, logger)
			processor := usecases.NewProcessor(l.validator, p, logger)

			var cmd usecases.HandleCommand
			cmd.ClientId = getStringMessageAttribute(msg, "client_id")
			cmd.EventId = getStringMessageAttribute(msg, "event_id")
			cmd.EventType = getStringMessageAttribute(msg, "event_type")

			logger.AddAttribute("client_id", cmd.ClientId)
			logger.AddAttribute("event_id", cmd.EventId)
			logger.AddAttribute("event_type", cmd.EventType)

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
				messageAttributes := map[string]types.MessageAttributeValue{}
				for k, v := range msg.MessageAttributes {
					messageAttributes[k] = types.MessageAttributeValue{
						DataType:         &v.DataType,
						BinaryListValues: v.BinaryListValues,
						BinaryValue:      v.BinaryValue,
						StringListValues: v.StringListValues,
						StringValue:      v.StringValue,
					}
				}
				_, err := l.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
					QueueUrl:          aws.String(l.dlqURL),
					MessageBody:       aws.String(message.Body),
					MessageAttributes: messageAttributes,
				})

				if err != nil {
					logger.AddAttribute("error_sending_to_dlq", err.Error())
					mu.Lock()
					batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
						ItemIdentifier: message.MessageId,
					})
					mu.Unlock()
				}
				logger.AddAttribute("sent_to_dlq", true)
			}
		}(message)
	}

	wg.Wait()

	return events.SQSEventResponse{
		BatchItemFailures: batchItemFailures,
	}, nil
}

func NewLambdaEntryPoint(
	dynamoClient *dynamodb.Client,
	eventsTableName string,
	dlqURL string,
	validator ports.Validator,
	sqsClient *sqs.Client,
) *LambdaEntryPoint {
	if dynamoClient == nil {
		panic("dynamo client is nil")
	}

	if sqsClient == nil {
		panic("sqs client is nil")
	}

	return &LambdaEntryPoint{
		dynamoClient:    dynamoClient,
		eventsTableName: eventsTableName,
		validator:       validator,
		sqsClient:       sqsClient,
	}
}
