package redisx

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type PubSubClient struct {
	client redis.UniversalClient
}

func NewPubSubClient(client redis.UniversalClient) *PubSubClient {
	return &PubSubClient{
		client: client,
	}
}

// Publish sends a message to the channel and forgets what it sent
// error returns only if redis errored, not if nobody is not listening

func (p *PubSubClient) Publish(ctx context.Context, channel, message string) error {
	if err := p.client.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("PubSubClient.Publish: %w", err)
	}
	return nil
}

// Subscribe creates a subscription to the given channels.
// IMPORTANT: Context propagation behavior:
// - The returned *redis.PubSub is independent from ctx cancellation
// - Caller MUST call pubSub.Close() to stop the subscription
// - Context is used only for the initial subscription call, not for the message channel lifetime
// - Goroutine receiving messages must check for context cancellation externally
//
// This prevents automatic goroutine leaks on shutdown. The caller is responsible for
// closing the PubSub subscription in response to context cancellation.
func (p *PubSubClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return p.client.Subscribe(ctx, channels...)
}
