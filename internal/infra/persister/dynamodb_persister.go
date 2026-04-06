package persister

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/guilhermeais/event-processor/internal/observability"
	"github.com/guilhermeais/event-processor/internal/ports"
)

type DynamoDbAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type DynamoPersister struct {
	db        DynamoDbAPI
	tableName string
	now       func() time.Time
	logger    *observability.Logger
}

func (p *DynamoPersister) Save(ctx context.Context, cmd ports.SaveCommand) error {
	item := map[string]types.AttributeValue{
		"id":         &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", cmd.ClientID, cmd.EventID)},
		"client_id":  &types.AttributeValueMemberS{Value: cmd.ClientID},
		"event_id":   &types.AttributeValueMemberS{Value: cmd.EventID},
		"event_type": &types.AttributeValueMemberS{Value: cmd.EventType},
		"payload":    &types.AttributeValueMemberS{Value: string(cmd.Payload)},
		"created_at": &types.AttributeValueMemberS{Value: cmd.CreatedAt.UTC().Format(time.RFC3339Nano)},
	}
	_, err := p.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(p.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err == nil {
		return nil
	}
	p.logger.AddAttribute("dynamodb_error", err.Error())

	// Error when the idempotency key is duplicated (client_id/event_id)
	var cfe *types.ConditionalCheckFailedException
	isIdempotencyError := errors.As(err, &cfe)
	if isIdempotencyError {
		p.logger.AddAttribute("is_idempotency_error", true)
		return nil
	}

	if isRetryableDynamoErr(err) {
		return fmt.Errorf("%w: %v", ports.ErrRetryable, err)
	}

	return err
}

func isRetryableDynamoErr(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout()) {
		return true
	}

	var pte *types.ProvisionedThroughputExceededException
	if errors.As(err, &pte) {
		return true
	}

	var rle *types.RequestLimitExceeded
	if errors.As(err, &rle) {
		return true
	}

	var te *types.ThrottlingException
	if errors.As(err, &te) {
		return true
	}

	var ise *types.InternalServerError
	if errors.As(err, &ise) {
		return true
	}

	return false
}

func NewDynamoPersister(db DynamoDbAPI, tableName string, logger *observability.Logger) *DynamoPersister {
	if logger == nil {
		panic("logger cannot be nil")
	}
	return &DynamoPersister{
		db:        db,
		tableName: tableName,
		now:       time.Now,
		logger:    logger,
	}
}
