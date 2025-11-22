package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskqueue/internal/broker"
	"taskqueue/pkg/metrics"
)

// Queue implements broker.Broker with in-memory slices (development only).
type Queue struct {
	mu       sync.RWMutex
	messages []*broker.Message
	stream   string
	notify   chan struct{}
}

// New creates a new memory broker for the provided stream.
func New(stream ...string) *Queue {
	name := "job_stream"
	if len(stream) > 0 && stream[0] != "" {
		name = stream[0]
	}
	return &Queue{
		stream:   name,
		notify:   make(chan struct{}, 1),
		messages: make([]*broker.Message, 0),
	}
}

// Enqueue pushes a job to the queue.
func (q *Queue) Enqueue(ctx context.Context, msg *broker.Message) error {
	q.mu.Lock()
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("mem-%s", uuid.NewString())
	}
	msg.EnqueuedAt = time.Now().UTC()
	q.messages = append(q.messages, msg)
	depth := len(q.messages)
	q.mu.Unlock()

	select {
	case q.notify <- struct{}{}:
	default:
	}
	metrics.QueueDepthGauge.WithLabelValues(q.stream).Set(float64(depth))
	return nil
}

// Consume pops messages FIFO, blocking until available or context cancelled.
func (q *Queue) Consume(ctx context.Context, stream, consumerID string, count int, block time.Duration) ([]*broker.Message, error) {
	if block <= 0 {
		block = time.Second
	}
	timer := time.NewTimer(block)
	defer timer.Stop()

	for {
		q.mu.Lock()
		total := len(q.messages)
		if total > 0 {
			if count <= 0 || count > total {
				count = total
			}
			msgs := q.messages[:count]
			q.messages = append([]*broker.Message{}, q.messages[count:]...)
			remaining := len(q.messages)
			q.mu.Unlock()
			metrics.QueueDepthGauge.WithLabelValues(q.stream).Set(float64(remaining))
			return msgs, nil
		}
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, nil
		case <-q.notify:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(block)
		}
	}
}

// Ack is a no-op for the in-memory broker.
func (q *Queue) Ack(ctx context.Context, stream, messageID string) error {
	return nil
}

// Requeue adds the message back to the queue with optional delay.
func (q *Queue) Requeue(ctx context.Context, stream string, msg *broker.Message, delay time.Duration) error {
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return q.Enqueue(ctx, msg)
}

// SendToDeadLetter simply drops the message (logs would be better in production).
func (q *Queue) SendToDeadLetter(ctx context.Context, stream string, msg *broker.Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}
	// For dev we just acknowledge the drop; production implementation would persist.
	return nil
}

// Depth returns the approximate queue length.
func (q *Queue) Depth() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.messages)
}

