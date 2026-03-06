package persister_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/guilhermeais/event-processor/internal/infra/persister"
	"github.com/guilhermeais/event-processor/internal/ports"
	testhelpers "github.com/guilhermeais/event-processor/tests/unit/testhelpers"
	"github.com/stretchr/testify/assert"
)

type Event struct {
	ClientID  string `dynamodbav:"client_id"`
	EventID   string `dynamodbav:"event_id"`
	EventType string `dynamodbav:"event_type"`
	Payload   string `dynamodbav:"payload"`
	CreatedAt string `dynamodbav:"created_at"`
}

func TestDynamoPersister_Save(t *testing.T) {
	t.Run("should save an item correctly", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cfg, cleanup := testhelpers.SetupLocalStack(ctx)
		defer cleanup()

		dynamoClient := dynamodb.NewFromConfig(cfg)

		// The idea is to crete a table for each test, enabling the paralelism on tests
		tableName := fmt.Sprintf("events_%s", uuid.NewString())
		err := testhelpers.CreateEventsTable(ctx, dynamoClient, tableName)
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

		key := map[string]types.AttributeValue{
			"client_id": &types.AttributeValueMemberS{Value: "client-1"},
			"event_id":  &types.AttributeValueMemberS{Value: "event-1"},
		}

		var event Event
		saved, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: &tableName,
			Key:       key,
		})
		err = attributevalue.UnmarshalMap(saved.Item, &event)
		assert.Nil(t, err)

		assert.Equal(t, event.ClientID, "client-1")
		assert.Equal(t, event.EventID, "event-1")
		assert.Equal(t, event.EventType, "event-type-1")
		assert.Equal(t, event.Payload, `{"id":"1"}`)
	})
}
