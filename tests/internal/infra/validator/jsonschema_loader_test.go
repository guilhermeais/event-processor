package validator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
	"github.com/guilhermeais/event-processor/internal/ports"
	"github.com/guilhermeais/event-processor/tests/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestDynamoDbJSONSchemaLoader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, cleanup := testhelpers.SetupLocalStack(ctx)
	defer cleanup()
	dynamoClient := dynamodb.NewFromConfig(cfg)
	t.Run("should load json schemas from dynamodb", func(t *testing.T) {
		tableName := fmt.Sprintf("schemas_%s", uuid.NewString())
		err := testhelpers.CreateSchemasTable(ctx, dynamoClient, tableName)
		assert.Nil(t, err)

		defer func() {
			testhelpers.DeleteTable(ctx, dynamoClient, tableName)
		}()

		dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &tableName,
			Item: map[string]types.AttributeValue{
				"event_type": &types.AttributeValueMemberS{Value: "an-event-type"},
				"schema": &types.AttributeValueMemberS{Value: `{
				"type":"object",
				"properties": {
					"id": { "type": "string" },
					"birthday": { "type": "string", "format": "date" }
				}
			}`},
			},
		})

		sut := validator.NewDynamoDbJSONSchemaLoader(dynamoClient, tableName)
		schemas, err := sut.Load(ctx)
		assert.Nil(t, err)

		assert.Len(t, schemas, 1)
		jsonSchemaValidator, err := validator.NewJSONSchemaValidator(schemas)
		assert.Nil(t, err)

		err = jsonSchemaValidator.Validate(
			ctx,
			"an-event-type",
			[]byte(`{"id":"123432","birthday": "2026-08-26"}`),
		)
		assert.Nil(t, err)

		err = jsonSchemaValidator.Validate(
			ctx,
			"an-event-type",
			[]byte(`{"id":"123432"`),
		)
		assert.ErrorIs(t, err, ports.ErrInvalidPayload)
	})
}
