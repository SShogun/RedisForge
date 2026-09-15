package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	streamMaxLen      = 100_000
	pendingIdleThresh = 30 * time.Second
)

type StreamClient struct {
	client redis.UniversalClient
}

func NewStreamClient(client redis.UniversalClient) *StreamClient {
	return &StreamClient{client: client}
}

// Append adds a message to a stream with approximate length trimming.
func (s *StreamClient) Append(ctx context.Context, stream string, fields map[string]interface{}) (string, error) {
	id, err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: streamMaxLen,
		Approx: true,
		Values: fields,
	}).Result()
	if err != nil {
		return "", fmt.Errorf("StreamClient.Append: %w", err)
	}
	return id, nil
}

// EnsureGroup creates a consumer group if it does not exist.
// The group starts at 0 so existing entries are replayed instead of silently skipped.
func (s *StreamClient) EnsureGroup(ctx context.Context, stream, group string) error {
	err := s.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("StreamClient.EnsureGroup: %w", err)
	}
	return nil
}

// Message wraps redis.XMessage with ACK metadata.
type Message struct {
	xm     redis.XMessage
	stream string
	group  string
	client redis.UniversalClient
}

func (m Message) ID() string {
	return m.xm.ID
}

func (m Message) Values() map[string]interface{} {
	return m.xm.Values
}

func (m Message) Ack(ctx context.Context) error {
	acked, err := m.client.XAck(ctx, m.stream, m.group, m.xm.ID).Result()
	if err != nil {
		return fmt.Errorf("Message.Ack: %w", err)
	}
	if acked != 1 {
		return fmt.Errorf("Message.Ack: expected 1 acknowledged entry, got %d", acked)
	}
	return nil
}

// ReadGroup reads new messages for a consumer up to count.
func (s *StreamClient) ReadGroup(
	ctx context.Context,
	stream, group, consumer string,
	count int64,
	blockDur time.Duration,
) ([]Message, error) {
	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    blockDur,
		NoAck:    false,
	}).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("StreamClient.ReadGroup: %w", err)
	}

	var msgs []Message
	for _, streamResult := range streams {
		for _, xm := range streamResult.Messages {
			msgs = append(msgs, Message{
				xm:     xm,
				stream: stream,
				group:  group,
				client: s.client,
			})
		}
	}
	return msgs, nil
}

// ClaimStale uses XAUTOCLAIM to transfer pending entries that have been idle
// longer than pendingIdleThresh. startID must be the cursor returned by the
// previous call; use "0-0" to begin a scan. Redis returns "0-0" when the scan
// has reached the end of the pending-entry list.
func (s *StreamClient) ClaimStale(
	ctx context.Context,
	stream, group, consumer, startID string,
	count int64,
) ([]Message, string, error) {
	if startID == "" {
		startID = "0-0"
	}
	if count <= 0 {
		return nil, "", fmt.Errorf("StreamClient.ClaimStale: count must be positive")
	}

	xmsgs, nextStartID, err := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  pendingIdleThresh,
		Start:    startID,
		Count:    count,
	}).Result()
	if err != nil {
		return nil, "", fmt.Errorf("StreamClient.ClaimStale: %w", err)
	}

	msgs := make([]Message, 0, len(xmsgs))
	for _, xm := range xmsgs {
		msgs = append(msgs, Message{
			xm:     xm,
			stream: stream,
			group:  group,
			client: s.client,
		})
	}
	return msgs, nextStartID, nil
}

// GetPendingCount returns the total number of pending messages for a consumer group.
func (s *StreamClient) GetPendingCount(ctx context.Context, stream, group string) (int64, error) {
	pending, err := s.client.XPending(ctx, stream, group).Result()
	if err != nil {
		return 0, fmt.Errorf("StreamClient.GetPendingCount: %w", err)
	}
	return pending.Count, nil
}
