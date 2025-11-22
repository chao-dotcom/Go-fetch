package broker

import (
	"context"
	"encoding/json"
	"time"
)

// Message represents a queue entry.
type Message struct {
	ID          string          `json:"id"`
	JobID       string          `json:"job_id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	Priority    int             `json:"priority"`
	EnqueuedAt  time.Time       `json:"enqueued_at"`
}

// Broker defines streaming queue operations.
type Broker interface {
	Enqueue(ctx context.Context, msg *Message) error
	Consume(ctx context.Context, stream string, consumerID string, count int, block time.Duration) ([]*Message, error)
	Ack(ctx context.Context, stream string, messageID string) error
	Requeue(ctx context.Context, stream string, msg *Message, delay time.Duration) error
	SendToDeadLetter(ctx context.Context, stream string, msg *Message) error
}

