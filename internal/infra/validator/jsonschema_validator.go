package validator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/guilhermeais/event-processor/internal/ports"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type schemas map[string]*jsonschema.Schema
type JSONSchemaValidator struct {
	schemas schemas
}

func (j *JSONSchemaValidator) Validate(ctx context.Context, eventType string, payload []byte) error {
	schema, exists := j.schemas[eventType]
	if !exists {
		return fmt.Errorf("%w: error type %s is invalid", ports.ErrInvalidPayload, eventType)
	}

	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		return fmt.Errorf("%w: %s", ports.ErrInvalidPayload, err.Error())
	}

	if err := schema.Validate(v); err != nil {
		return fmt.Errorf("%w: %s", ports.ErrInvalidPayload, err.Error())
	}

	return nil
}

type UncompiledSchemas struct {
	EventType string
	Schema    string
}

func NewValidatorFromReader(uncompiledSchemas []UncompiledSchemas) (*JSONSchemaValidator, error) {
	schemas := schemas{}
	for _, s := range uncompiledSchemas {
		schema, err := jsonschema.CompileString(s.EventType, s.Schema)
		if err != nil {
			return nil, err
		}

		schemas[s.EventType] = schema
	}

	return &JSONSchemaValidator{schemas: schemas}, nil
}
