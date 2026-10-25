package bootstrap

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
	"github.com/guilhermeais/event-processor/internal/ports"
)

type ProcessingDependenciesManager struct {
	DynamoClient    *dynamodb.Client
	EventsTableName string
	DLQUrl          string
	Validator       ports.Validator
	SchemaLoader    validator.JSONSchemaLoader
	SqsClient       *sqs.Client
	awsConfig       aws.Config
	config          Config
}

type Config struct {
	EventsTableName string
	DLQUrl          string
	SchemaTableName string
}

func NewProcessingDependenciesManager(
	ctx context.Context,
	awsConfig aws.Config,
	config Config,
) *ProcessingDependenciesManager {
	dynamoClient := dynamodb.NewFromConfig(awsConfig)
	sqsClient := sqs.NewFromConfig(awsConfig)
	l := validator.NewDynamoDbJSONSchemaLoader(dynamoClient, config.SchemaTableName)

	return &ProcessingDependenciesManager{
		awsConfig:       awsConfig,
		config:          config,
		DynamoClient:    dynamoClient,
		SqsClient:       sqsClient,
		EventsTableName: config.EventsTableName,
		DLQUrl:          config.DLQUrl,
		SchemaLoader:    l,
	}
}

func (p *ProcessingDependenciesManager) LoadDependencies(
	ctx context.Context,
) (*ProcessingDependenciesManager, error) {
	schemas, err := p.SchemaLoader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("error on loading schemas: %v", err)
	}
	v, err := validator.NewJSONSchemaValidator(schemas)
	if err != nil {
		return nil, fmt.Errorf("error on creating validator: %v", err)
	}
	p.Validator = v
	return p, nil
}
