package validator_test

import (
	"context"
	"testing"

	"github.com/guilhermeais/event-processor/internal/infra/validator"
	"github.com/guilhermeais/event-processor/internal/ports"
	"github.com/stretchr/testify/assert"
)

func makeSut(t *testing.T) *validator.JSONSchemaValidator {
	t.Helper()
	v, err := validator.NewJSONSchemaValidator([]validator.UncompiledSchema{
		{
			EventType: "test",
			Schema: `{
				"type":"object",
				"properties": {
					"id": { "type": "string" },
					"birthday": { "type": "string", "format": "date" }
				}
			}`,
		},
	})
	assert.Nil(t, err)
	return v
}

func TestJSONSchemaValidator(t *testing.T) {
	t.Run("should return error when invalid event type", func(t *testing.T) {
		sut := makeSut(t)
		err := sut.Validate(context.Background(), "invalid-event-type", []byte(`{"id":"123432","birthday": "2026-08-26"}`))
		assert.ErrorIs(t, err, ports.ErrInvalidPayload)
	})
	t.Run("should return not return error when provide a valid schema", func(t *testing.T) {
		sut := makeSut(t)
		err := sut.Validate(context.Background(), "test", []byte(`{"id":"123432","birthday": "2026-08-26"}`))
		assert.Nil(t, err)
	})
}
