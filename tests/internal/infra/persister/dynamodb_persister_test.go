package persister_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/guilhermeais/event-processor/internal/infra/persister"
	"github.com/guilhermeais/event-processor/internal/ports"
	testhelpers "github.com/guilhermeais/event-processor/tests/testhelpers"
	"github.com/stretchr/testify/assert"
)

type EventDynamodb struct {
	ClientID  string `dynamodbav:"client_id"`
	EventID   string `dynamodbav:"event_id"`
	EventType string `dynamodbav:"event_type"`
	Payload   string `dynamodbav:"payload"`
	CreatedAt string `dynamodbav:"created_at"`
}

func getDynamoDbEvent(
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

func TestDynamoPersister_Save(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, cleanup := testhelpers.SetupLocalStack(ctx)
	defer cleanup()
	dynamoClient := dynamodb.NewFromConfig(cfg)
	t.Run("should save an item correctly", func(t *testing.T) {
		// The idea is to crete a table for each test, enabling the paralelism on tests
		tableName := fmt.Sprintf("events_%s", uuid.NewString())
		err := testhelpers.CreateEventsTable(ctx, dynamoClient, tableName)
		assert.Nil(t, err)
		defer func() {
			testhelpers.DeleteTable(ctx, dynamoClient, tableName)
		}()

		sut := persister.NewDynamoPersister(dynamoClient, tableName)
		err = sut.Save(ctx, ports.SaveCommand{
			ClientID:  "client-1",
			EventID:   "event-1",
			EventType: "event-type-1",
			Payload:   []byte(`{"id":"1"}`),
			CreatedAt: time.Now(),
		})
		assert.Nil(t, err)
		event := getDynamoDbEvent(
			t,
			ctx,
			dynamoClient,
			tableName,
			"client-1",
			"event-1",
		)

		assert.Equal(t, event.ClientID, "client-1")
		assert.Equal(t, event.EventID, "event-1")
		assert.Equal(t, event.EventType, "event-type-1")
		assert.Equal(t, event.Payload, `{"id":"1"}`)
	})

	t.Run("should not save duplicated item (same client_id/event_id)", func(t *testing.T) {
		tableName := fmt.Sprintf("events_%s", uuid.NewString())
		err := testhelpers.CreateEventsTable(ctx, dynamoClient, tableName)
		defer func() {
			testhelpers.DeleteTable(ctx, dynamoClient, tableName)
		}()

		assert.Nil(t, err)

		sut := persister.NewDynamoPersister(dynamoClient, tableName)
		err = sut.Save(ctx, ports.SaveCommand{
			ClientID:  "client-1",
			EventID:   "event-1",
			EventType: "event-type-1",
			Payload:   []byte(`{"id":"1"}`),
			CreatedAt: time.Now(),
		})
		assert.Nil(t, err)

		err = sut.Save(ctx, ports.SaveCommand{
			ClientID:  "client-1",
			EventID:   "event-1",
			EventType: "event-type-1",
			Payload:   []byte(`{"id":"1651"}`),
			CreatedAt: time.Now(),
		})
		assert.Nil(t, err)

		event := getDynamoDbEvent(
			t,
			ctx,
			dynamoClient,
			tableName,
			"client-1",
			"event-1",
		)

		assert.Equal(t, event.ClientID, "client-1")
		assert.Equal(t, event.EventID, "event-1")
		assert.Equal(t, event.EventType, "event-type-1")
		assert.Equal(t, event.Payload, `{"id":"1"}`) // keep the first payload
	})

	t.Run("should save same event id but to different client id", func(t *testing.T) {
		tableName := fmt.Sprintf("events_%s", uuid.NewString())
		err := testhelpers.CreateEventsTable(ctx, dynamoClient, tableName)
		defer func() {
			testhelpers.DeleteTable(ctx, dynamoClient, tableName)
		}()

		assert.Nil(t, err)

		sut := persister.NewDynamoPersister(dynamoClient, tableName)
		err = sut.Save(ctx, ports.SaveCommand{
			ClientID:  "client-1",
			EventID:   "event-1",
			EventType: "event-type-1",
			Payload:   []byte(`{"id":"1"}`),
			CreatedAt: time.Now(),
		})
		assert.Nil(t, err)

		err = sut.Save(ctx, ports.SaveCommand{
			ClientID:  "client-2",
			EventID:   "event-1",
			EventType: "event-type-1",
			Payload:   []byte(`{"id":"1651"}`),
			CreatedAt: time.Now(),
		})
		assert.Nil(t, err)

		firstClientEvent := getDynamoDbEvent(
			t,
			ctx,
			dynamoClient,
			tableName,
			"client-1",
			"event-1",
		)

		assert.Equal(t, firstClientEvent.ClientID, "client-1")
		assert.Equal(t, firstClientEvent.EventID, "event-1")
		assert.Equal(t, firstClientEvent.EventType, "event-type-1")
		assert.Equal(t, firstClientEvent.Payload, `{"id":"1"}`)

		secondClientEvent := getDynamoDbEvent(
			t,
			ctx,
			dynamoClient,
			tableName,
			"client-2",
			"event-1",
		)

		assert.Equal(t, secondClientEvent.ClientID, "client-2")
		assert.Equal(t, secondClientEvent.EventID, "event-1")
		assert.Equal(t, secondClientEvent.EventType, "event-type-1")
		assert.Equal(t, secondClientEvent.Payload, `{"id":"1651"}`)

	})

	t.Run("should return non retryable error if table does not exists", func(t *testing.T) {
		tableName := fmt.Sprintf("events_%s", uuid.NewString())

		sut := persister.NewDynamoPersister(dynamoClient, tableName)
		err := sut.Save(ctx, ports.SaveCommand{
			ClientID:  "client-1",
			EventID:   "event-1",
			EventType: "event-type-1",
			Payload:   []byte(`{"id":"1"}`),
			CreatedAt: time.Now(),
		})
		assert.NotErrorIs(t, err, ports.ErrRetryable)
	})
}

func TestDynamoPersister_Save_RetryableErros(
	t *testing.T,
) {
	ctx := context.Background()

	testCases := []struct {
		name            string
		mockError       error
		expectRetryable bool
	}{
		{
			name:            "ProvisionedThroughputExceededException",
			mockError:       &types.ProvisionedThroughputExceededException{Message: aws.String("throttled")},
			expectRetryable: true,
		},
		{
			name:            "ThrottlingException",
			mockError:       &types.ThrottlingException{Message: aws.String("throttled")},
			expectRetryable: true,
		},
		{
			name:            "InternalServerError",
			mockError:       &types.InternalServerError{Message: aws.String("internal")},
			expectRetryable: true,
		},
		{
			name:            "RequestLimitExceeded",
			mockError:       &types.RequestLimitExceeded{Message: aws.String("limit")},
			expectRetryable: true,
		},
		{
			name:            "Context DeadlineExceeded",
			mockError:       context.DeadlineExceeded,
			expectRetryable: true,
		},
		{
			name:            "ResourceNotFoundException - non retryable",
			mockError:       &types.ResourceNotFoundException{Message: aws.String("not found")},
			expectRetryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &testhelpers.MockDynamoDB{
				PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return nil, tc.mockError
				},
			}

			sut := persister.NewDynamoPersister(mock, "test-table")
			err := sut.Save(ctx, ports.SaveCommand{
				ClientID:  "client-1",
				EventID:   "event-1",
				EventType: "test",
				Payload:   []byte(`{}`),
				CreatedAt: time.Now(),
			})

			if tc.expectRetryable {
				assert.ErrorIs(t, err, ports.ErrRetryable)
			} else {
				assert.NotErrorIs(t, err, ports.ErrRetryable)
				assert.Error(t, err)
			}
		})
	}
}
