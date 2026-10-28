package bootstrap_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/guilhermeais/event-processor/internal/bootstrap"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
	"github.com/guilhermeais/event-processor/tests/testhelpers"
	"github.com/stretchr/testify/assert"
)

type JSONSchemaLoaderMock struct {
	MockedError  error
	MockedResult []validator.UncompiledSchema
}

func (s *JSONSchemaLoaderMock) Load(ctx context.Context) ([]validator.UncompiledSchema, error) {
	return s.MockedResult, s.MockedError
}

func TestProcessingDependenciesManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, cleanup := testhelpers.SetupLocalStack(ctx)
	defer cleanup()

	dynamo := dynamodb.NewFromConfig(cfg)
	sqsClientLocal := sqs.NewFromConfig(cfg)

	t.Run("should return an error when loading schemas fail", func(t *testing.T) {
		mockedError := fmt.Errorf("unexpecte error")
		expectedError := fmt.Errorf("error on loading schemas: %v", mockedError)
		mockedSchemaLoader := &JSONSchemaLoaderMock{
			MockedError: mockedError,
		}
		deps := testhelpers.PrepareDependencies(
			ctx,
			t,
			dynamo,
			sqsClientLocal,
		)

		pdm := bootstrap.NewProcessingDependenciesManager(
			ctx, cfg, bootstrap.Config{
				EventsTableName: deps.EventsTbl,
				SchemaTableName: deps.SchemasTbl,
				DLQUrl:          deps.DLQUrl,
			},
		)
		pdm.SchemaLoader = mockedSchemaLoader
		_, err := pdm.LoadDependencies(ctx)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), expectedError.Error())
	})
	t.Run("should return an error when compiling schemas fail", func(t *testing.T) {
		mockedSchemaLoader := &JSONSchemaLoaderMock{
			MockedResult: []validator.UncompiledSchema{
				{
					EventType: "invalid-schema",
					Schema:    "{",
				},
			},
		}
		deps := testhelpers.PrepareDependencies(
			ctx,
			t,
			dynamo,
			sqsClientLocal,
		)

		pdm := bootstrap.NewProcessingDependenciesManager(
			ctx, cfg, bootstrap.Config{
				EventsTableName: deps.EventsTbl,
				SchemaTableName: deps.SchemasTbl,
				DLQUrl:          deps.DLQUrl,
			},
		)
		pdm.SchemaLoader = mockedSchemaLoader
		_, err := pdm.LoadDependencies(ctx)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "error on creating validator: jsonschema: invalid json invalid-schema: unexpected EOF")
	})
	t.Run("should return all dependencies correctly", func(t *testing.T) {
		deps := testhelpers.PrepareDependencies(
			ctx,
			t,
			dynamo,
			sqsClientLocal,
		)

		pdm := bootstrap.NewProcessingDependenciesManager(
			ctx, cfg, bootstrap.Config{
				EventsTableName: deps.EventsTbl,
				SchemaTableName: deps.SchemasTbl,
				DLQUrl:          deps.DLQUrl,
			},
		)
		_, err := pdm.LoadDependencies(ctx)
		assert.NoError(t, err)
		assert.Equal(t, pdm.DLQUrl, deps.DLQUrl)
		assert.Equal(t, pdm.EventsTableName, deps.EventsTbl)
		assert.NotNil(t, pdm.DynamoClient)
		fakeId := map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "fake-id"},
		}
		dbItem, err := pdm.DynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: &pdm.EventsTableName,
			Key:       fakeId,
		})
		assert.NoError(t, err)
		assert.NotNil(t, dbItem)
		assert.Nil(t, dbItem.Item)
		assert.NotNil(t, pdm.SqsClient)
		queues, err := pdm.SqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
		assert.NoError(t, err)
		assert.NotNil(t, queues)
		assert.Contains(t, queues.QueueUrls, deps.DLQUrl)
	})
}
