package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	b "github.com/guilhermeais/event-processor/internal/bootstrap"
	"github.com/guilhermeais/event-processor/internal/infra/entrypoint"
	"github.com/guilhermeais/event-processor/internal/infra/enviroment"
	"github.com/guilhermeais/event-processor/internal/observability"
)

func bootstrap(ctx context.Context) (*entrypoint.LambdaEntryPoint, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}
	pdm := b.NewProcessingDependenciesManager(ctx, cfg, b.Config{
		EventsTableName: enviroment.EnvironmentVariables.SCHEMA_TABLE_NAME,
		SchemaTableName: enviroment.EnvironmentVariables.SCHEMA_TABLE_NAME,
		DLQUrl:          enviroment.EnvironmentVariables.DLQ_URL,
	})
	deps, err := pdm.LoadDependencies(ctx)
	return entrypoint.NewLambdaEntryPoint(
		deps.DynamoClient,
		deps.EventsTableName,
		deps.DLQUrl,
		deps.Validator,
		deps.SqsClient,
		observability.NewLoggerDefault,
	), nil
}

func main() {
	ctx, subseg := xray.BeginSegment(context.Background(), "event-processor")
	defer subseg.Close(nil)
	lambdaHandler, err := bootstrap(ctx)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
	lambda.Start(lambdaHandler.Handler)
}
