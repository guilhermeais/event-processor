package ports

import "context"

type Validator interface {
	Validate(ctx context.Context, eventType string, payload []byte) error
}
