package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/guilhermeais/event-processor/internal/ports"
	"github.com/guilhermeais/event-processor/internal/usecases"
	testhelpers "github.com/guilhermeais/event-processor/tests/unit/testhelpers"
	"github.com/stretchr/testify/assert"
)

type StubValidator struct {
	Calls       []validatorCalls
	MockedError error
}

type validatorCalls struct {
	eventType string
	payload   []byte
}

func (v *StubValidator) Validate(ctx context.Context, eventType string, payload []byte) error {
	v.Calls = append(v.Calls, validatorCalls{
		eventType: eventType,
		payload:   payload,
	})
	return v.MockedError
}

type StubPersister struct {
	Calls       []ports.SaveCommand
	MockedError error
}

func (p *StubPersister) Save(ctx context.Context, cmd ports.SaveCommand) error {
	p.Calls = append(p.Calls, cmd)
	return p.MockedError
}

func makeSut(t *testing.T) (*usecases.Processor, *StubValidator, *StubPersister) {
	validator := &StubValidator{}
	persister := &StubPersister{}
	logger, _ := testhelpers.CreateLogger(t)

	processor := usecases.NewProcessor(
		validator,
		persister,
		logger,
	)
	return processor, validator, persister
}

func TestProcessor(t *testing.T) {
	t.Run("should return DecisionToDLQ when validation fails", func(t *testing.T) {
		sut, validator, persister := makeSut(t)
		validator.MockedError = errors.New("invalid payload")
		result, err := sut.Handle(context.Background(), usecases.HandleCommand{
			ClientId:  "client_id_123",
			EventId:   "123321",
			EventType: "create-order",
			Payload:   []byte{},
		})
		assert.Equal(t, result, usecases.DecisionToDLQ)
		assert.Len(t, persister.Calls, 0)
		assert.EqualError(t, err, validator.MockedError.Error())
	})

	t.Run("should return DecisionRetry when persister fails but return a retryable error", func(t *testing.T) {
		sut, _, persister := makeSut(t)
		persister.MockedError = fmt.Errorf("%w: connection error", ports.ErrRetryable)
		result, err := sut.Handle(context.Background(), usecases.HandleCommand{
			ClientId:  "client_id_123",
			EventId:   "123321",
			EventType: "create-order",
			Payload:   []byte{},
		})
		assert.Equal(t, result, usecases.DecisionRetry)
		assert.Len(t, persister.Calls, 1)
		assert.EqualError(t, err, persister.MockedError.Error())
	})

	t.Run("should return DecisionToDLQ when persister fails", func(t *testing.T) {
		sut, _, persister := makeSut(t)
		persister.MockedError = errors.New("an error")
		result, err := sut.Handle(context.Background(), usecases.HandleCommand{
			ClientId:  "client_id_123",
			EventId:   "123321",
			EventType: "create-order",
			Payload:   []byte{},
		})
		assert.Equal(t, result, usecases.DecisionToDLQ)
		assert.Len(t, persister.Calls, 1)
		assert.EqualError(t, err, persister.MockedError.Error())
	})

	t.Run("should return DecisionAck when persist the event", func(t *testing.T) {
		sut, validator, persister := makeSut(t)
		result, err := sut.Handle(context.Background(), usecases.HandleCommand{
			ClientId:  "client_id_123",
			EventId:   "123321",
			EventType: "create-order",
			Payload:   []byte{},
		})
		assert.Nil(t, err)
		assert.Equal(t, result, usecases.DecisionAck)
		assert.Len(t, validator.Calls, 1)
		assert.Len(t, persister.Calls, 1)
	})
}
