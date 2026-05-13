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
	return &StreamClient{
		client: client,
	}
}

// append a message to stream with approx length trimming
// fields is a flat key-value slice: []interface{}{"event", "created", "id", "123"}

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

// EnsureGroup created a consumer group if it doesnt exist
// $ used to start from new messages

func (s *StreamClient) EnsureGroup(ctx context.Context, stream, group string) error {
	err := s.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("StreamClient.EnsureGroup: %w", err)
	}
	return nil
}

// wrapper around redis.XMessage with a helper
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
	return m.client.XAck(ctx, m.stream, m.group, m.xm.ID).Err()
}

// ReadGroup reads the new messages uptil a limit for a consumer
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
	for _, s2 := range streams {
		for _, xm := range s2.Messages {
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

// ClaimStale uses XAUTOCLAIM to steal pending entries idle for > 30s
// This handles dead consumers that crashed without ACKing.
// Returns the claimed messages and the next cursor (for pagination).
func (s *StreamClient) ClaimStale(
	ctx context.Context,
	stream, group, consumer string,
	count int64,
) ([]Message, string, error) {
	xmsgs, nextStartID, err := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  pendingIdleThresh,
		Start:    "0",
		Count:    count,
	}).Result()
	if err != nil {
		return nil, "", fmt.Errorf("StreamClient.ClaimStale: %w", err)
	}

	var msgs []Message
	for _, xm := range xmsgs {
		msgs = append(msgs, Message{
			xm: xm, stream: stream, group: group, client: s.client,
		})
	}
	return msgs, nextStartID, nil
}
