package bootstrap_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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

	t.Run("TODO: should return all dependencies correctly", func(t *testing.T) {
		t.Fail()
	})
}
