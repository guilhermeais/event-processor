package validator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type JSONSchemaLoader interface {
	Load(ctx context.Context) []UncompiledSchema
}

type DynamoDbJSONSchemaLoader struct {
	dynamodb  *dynamodb.Client
	tableName string
}

type UncompiledSchemaDynamoDb struct {
	EventType string `dynamodbav:"event_type"`
	Schema    string `dynamodbav:"schema"`
}

func (d *DynamoDbJSONSchemaLoader) Load(ctx context.Context) ([]UncompiledSchema, error) {
	results := []UncompiledSchema{}
	slog.Log(ctx, slog.LevelInfo, fmt.Sprintf("loading schemas from dynamodb table %s...", d.tableName))
	rawResults, err := d.dynamodb.Scan(ctx, &dynamodb.ScanInput{
		TableName: &d.tableName,
	})
	if err != nil {
		slog.Log(ctx, slog.LevelError, fmt.Sprintf("error searching schemas from dynamodb: %v", err))
		return nil, err
	}
	slog.Log(ctx, slog.LevelInfo, fmt.Sprintf("found %d schemas from dynamodb.", len(rawResults.Items)))

	for _, item := range rawResults.Items {
		var parsed UncompiledSchemaDynamoDb
		err = attributevalue.UnmarshalMap(item, &parsed)
		if err != nil {
			return nil, err
		}
		results = append(results, UncompiledSchema{
			EventType: parsed.EventType,
			Schema:    parsed.Schema,
		})
	}

	return results, nil
}

func NewDynamoDbJSONSchemaLoader(
	client *dynamodb.Client,
	tableName string,
) *DynamoDbJSONSchemaLoader {
	return &DynamoDbJSONSchemaLoader{
		dynamodb:  client,
		tableName: tableName,
	}
}
