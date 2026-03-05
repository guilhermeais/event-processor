package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/guilhermeais/event-processor/internal/observability"
	"github.com/guilhermeais/event-processor/internal/ports"
)

type Processor struct {
	validator ports.Validator
	persister ports.Persister
	logger    *observability.Logger
}

type HandleCommand struct {
	ClientId, EventId, EventType string
	Payload                      []byte
}

type HandleDecision string

const (
	DecisionAck   HandleDecision = "ACK"
	DecisionRetry HandleDecision = "RETRY"
	DecisionToDLQ HandleDecision = "TO_DLQ"
)

func (p Processor) Handle(ctx context.Context, cmd HandleCommand) (HandleDecision, error) {
	err := p.validator.Validate(ctx, cmd.EventType, cmd.Payload)
	if err != nil {
		p.logger.AddError(err)
		return DecisionToDLQ, err
	}

	err = p.persister.Save(ctx, ports.SaveCommand{
		ClientID:  cmd.ClientId,
		EventID:   cmd.EventId,
		EventType: cmd.EventType,
		Payload:   cmd.Payload,
		CreatedAt: time.Now(),
	})

	if err != nil {
		p.logger.AddError(err)
		if errors.Is(err, ports.ErrRetryable) {
			p.logger.AddAttribute("is_retryable", true)
			return DecisionRetry, err
		}
		return DecisionToDLQ, err
	}

	return DecisionAck, nil
}

func NewProcessor(
	validator ports.Validator,
	persister ports.Persister,
	logger *observability.Logger,
) *Processor {
	return &Processor{
		validator: validator,
		persister: persister,
		logger:    logger,
	}
}
