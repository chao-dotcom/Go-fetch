package redisbroker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"taskqueue/internal/broker"
)

const (
	defaultStream       = "job_stream"
	defaultGroup        = "workers"
	deadLetterSuffix    = ":dead"
	defaultMaxStreamLen = int64(10_000)
	defaultBatchSize    = 10
	defaultBlockTime    = 5 * time.Second
	claimMinIdleTime    = 30 * time.Second
)

// Broker implements broker.Broker backed by Redis Streams.
type Broker struct {
	client *redis.Client
	stream string
	group  string
	logger *zap.Logger
}

// New creates a Broker with the given Redis options, default stream and consumer group.
func New(ctx context.Context, opts *redis.Options, stream, group string, logger *zap.Logger) (*Broker, error) {
	if opts == nil {
		opts = &redis.Options{Addr: "127.0.0.1:6379"}
	}
	if stream == "" {
		stream = defaultStream
	}
	if group == "" {
		group = defaultGroup
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	b := &Broker{
		client: client,
		stream: stream,
		group:  group,
		logger: logger,
	}
	if err := b.ensureGroup(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Broker) ensureGroup(ctx context.Context) error {
	if b.group == "" {
		return nil
	}
	if err := b.client.XGroupCreateMkStream(ctx, b.stream, b.group, "0").Err(); err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return fmt.Errorf("redis: create group: %w", err)
		}
	}
	// Ensure DLQ stream exists too.
	deadStream := b.stream + deadLetterSuffix
	if err := b.client.XGroupCreateMkStream(ctx, deadStream, b.group, "0").Err(); err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			b.logger.Warn("redis: create DLQ group", zap.Error(err))
		}
	}
	return nil
}

// Close releases the Redis client.
func (b *Broker) Close() {
	if b.client != nil {
		_ = b.client.Close()
	}
}

// Client returns the underlying Redis client for metrics collection
func (b *Broker) Client() *redis.Client {
	return b.client
}

// Enqueue adds the message to the configured stream.
func (b *Broker) Enqueue(ctx context.Context, msg *broker.Message) error {
	msg.EnqueuedAt = time.Now().UTC()
	payload := msg.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	values := map[string]any{
		"job_id":       msg.JobID,
		"type":         msg.Type,
		"payload":      string(payload),
		"attempts":     msg.Attempts,
		"max_attempts": msg.MaxAttempts,
		"priority":     msg.Priority,
		"enqueued_at":  msg.EnqueuedAt.UnixNano(),
	}
	args := &redis.XAddArgs{
		Stream: b.stream,
		MaxLen: defaultMaxStreamLen,
		Approx: true,
		Values: values,
	}
	id, err := b.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("redis: enqueue: %w", err)
	}
	msg.ID = id
	return nil
}

// Consume reads messages from the stream.
func (b *Broker) Consume(ctx context.Context, stream string, consumerID string, count int, block time.Duration) ([]*broker.Message, error) {
	if stream == "" {
		stream = b.stream
	}
	if b.group == "" {
		return nil, fmt.Errorf("redis: consumer group is not configured")
	}
	if count <= 0 {
		count = defaultBatchSize
	}
	if block <= 0 {
		block = defaultBlockTime
	}

	// First try to reclaim abandoned messages
	if reclaimed, err := b.claimAbandonedMessages(ctx, consumerID, count); err == nil && len(reclaimed) > 0 {
		return reclaimed, nil
	} else if err != nil {
		b.logger.Warn("redis: claim abandoned", zap.Error(err))
	}

	args := &redis.XReadGroupArgs{
		Group:    b.group,
		Consumer: consumerID,
		Streams:  []string{stream, ">"},
		Count:    int64(count),
		Block:    block,
		NoAck:    false,
	}
	results, err := b.client.XReadGroup(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis: read group: %w", err)
	}
	var messages []*broker.Message
	for _, res := range results {
		for _, entry := range res.Messages {
			msg, convErr := b.convertEntry(entry)
			if convErr != nil {
				b.logger.Error("redis: parse message", zap.Error(convErr), zap.String("message_id", entry.ID))
				_ = b.client.XAck(ctx, stream, b.group, entry.ID).Err()
				continue
			}
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// Ack acknowledges a processed message.
func (b *Broker) Ack(ctx context.Context, stream, messageID string) error {
	if stream == "" {
		stream = b.stream
	}
	if messageID == "" || b.group == "" {
		return nil
	}
	if err := b.client.XAck(ctx, stream, b.group, messageID).Err(); err != nil {
		return fmt.Errorf("redis: ack: %w", err)
	}
	_ = b.client.XDel(ctx, stream, messageID).Err()
	return nil
}

// Requeue adds the message back to the stream after an optional delay.
func (b *Broker) Requeue(ctx context.Context, stream string, msg *broker.Message, delay time.Duration) error {
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return b.Enqueue(ctx, msg)
}

// SendToDeadLetter publishes failed messages to a dead-letter stream.
func (b *Broker) SendToDeadLetter(ctx context.Context, stream string, msg *broker.Message) error {
	target := stream
	if target == "" {
		target = b.stream
	}
	payload := msg.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	args := &redis.XAddArgs{
		Stream: target + deadLetterSuffix,
		MaxLen: defaultMaxStreamLen,
		Approx: true,
		Values: map[string]any{
			"job_id":   msg.JobID,
			"type":     msg.Type,
			"payload":  string(payload),
			"attempts": msg.Attempts,
		},
	}
	if _, err := b.client.XAdd(ctx, args).Result(); err != nil {
		return fmt.Errorf("redis: dead letter: %w", err)
	}
	return nil
}

func (b *Broker) claimAbandonedMessages(ctx context.Context, consumerID string, count int) ([]*broker.Message, error) {
	if count <= 0 {
		count = defaultBatchSize
	}
	pending, err := b.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: b.stream,
		Group:  b.group,
		Start:  "-",
		End:    "+",
		Count:  int64(count),
		Idle:   claimMinIdleTime,
	}).Result()
	if err != nil || len(pending) == 0 {
		return nil, err
	}

	ids := make([]string, 0, len(pending))
	for _, entry := range pending {
		ids = append(ids, entry.ID)
	}

	messages, err := b.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   b.stream,
		Group:    b.group,
		Consumer: consumerID,
		MinIdle:  claimMinIdleTime,
		Messages: ids,
	}).Result()
	if err != nil {
		return nil, err
	}

	var jobs []*broker.Message
	for _, entry := range messages {
		msg, convErr := b.convertEntry(entry)
		if convErr != nil {
			b.logger.Error("redis: parse claimed message", zap.Error(convErr), zap.String("message_id", entry.ID))
			continue
		}
		jobs = append(jobs, msg)
	}
	return jobs, nil
}

func (b *Broker) convertEntry(entry redis.XMessage) (*broker.Message, error) {
	jobID, ok := entry.Values["job_id"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing job_id")
	}
	jobType, ok := entry.Values["type"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing type")
	}

	payload := []byte{}
	switch v := entry.Values["payload"].(type) {
	case string:
		payload = []byte(v)
	case []byte:
		payload = v
	case nil:
	default:
		serialized, err := json.Marshal(v)
		if err == nil {
			payload = serialized
		}
	}

	msg := &broker.Message{
		ID:          entry.ID,
		JobID:       jobID,
		Type:        jobType,
		Payload:     payload,
		Attempts:    asInt(entry.Values["attempts"]),
		MaxAttempts: asInt(entry.Values["max_attempts"]),
		Priority:    asInt(entry.Values["priority"]),
		EnqueuedAt:  asTime(entry.Values["enqueued_at"]),
	}
	return msg, nil
}

func asInt(val any) int {
	switch v := val.(type) {
	case nil:
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

func asTime(val any) time.Time {
	switch v := val.(type) {
	case int64:
		if v == 0 {
			return time.Time{}
		}
		return time.Unix(0, v).UTC()
	case string:
		ns, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}
		}
		return time.Unix(0, ns).UTC()
	default:
		return time.Time{}
	}
}
