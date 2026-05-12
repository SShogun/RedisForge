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
	if err := p.client.Publish(ctx, channel, message); err != nil {
		return fmt.Errorf("PubSubClient.Publish: %w", err)
	}
	return nil
}

// Subscribe creates a sub to the given channels

func (p *PubSubClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return p.client.PSubscribe(ctx, channels...)
}
