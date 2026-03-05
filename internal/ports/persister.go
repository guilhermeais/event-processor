package ports

import (
	"context"
	"time"
)

type Persister interface {
	Save(ctx context.Context, cmd SaveCommand) error
}

type SaveCommand struct {
	ClientID  string
	EventID   string
	EventType string
	Payload   []byte
	CreatedAt time.Time
}
