package ports

import "context"

type Persister interface {
	Save(ctx context.Context, cmd SaveCommand) error
}

type Status string

const ReadyForDelivery = "READY_FOR_DELIVERY"

type SaveCommand struct {
	ClientID  string
	EventID   string
	EventType string
	Payload   []byte
	Status    Status
}
