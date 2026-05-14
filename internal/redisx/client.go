package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/SShogun/redisforge/internal/config"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type UniversalClient = redis.UniversalClient

// we use universal client since it works with all 3 - single, sentinel & cluster redis

func Open(ctx context.Context, cfg config.Redis) (redis.UniversalClient, error) {
	var client UniversalClient

	switch {
	case cfg.ClusterEnabled:
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.ClusterAddrs,
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			Protocol:     2,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})

	case cfg.SentinelEnabled:
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.SentinelMasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			PoolSize:      cfg.PoolSize,
			Protocol:      2,
			DialTimeout:   5 * time.Second,
			ReadTimeout:   3 * time.Second,
			WriteTimeout:  3 * time.Second,
		})
	default:
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			Protocol:     2,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
	}
	if err := redisotel.InstrumentTracing(client); err != nil {
		return nil, fmt.Errorf("redisx.Open: instrument tracing: %w", err)
	}

	// Validate connectivity.
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redisx.Open: ping: %w", err)
	}

	return client, nil
}
