package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/guilhermeais/event-processor/internal/infra/entrypoint"
	"github.com/guilhermeais/event-processor/internal/infra/validator"
)

func bootstrap(ctx context.Context) (*entrypoint.LambdaEntryPoint, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	schemaTable := os.Getenv("SCHEMA_TABLE_NAME")
	eventsTableName := os.Getenv("EVENTS_TABLE_NAME")
	dlqURL := os.Getenv("DLQ_URL")

	l := validator.NewDynamoDbJSONSchemaLoader(dynamoClient, schemaTable)
	schemas, err := l.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("error on loading schemas: %v", err)
	}
	v, err := validator.NewJSONSchemaValidator(schemas)
	if err != nil {
		return nil, fmt.Errorf("error on creating validator: %v", err)
	}
	return entrypoint.NewLambdaEntryPoint(dynamoClient, eventsTableName, dlqURL, v, sqsClient), nil
}

func main() {
	lambdaHandler, err := bootstrap(context.Background())
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
	lambda.Start(lambdaHandler.Handler)
}
