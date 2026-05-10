package config

import (
	"fmt"
	"time"

	env "github.com/caarlos0/env/v10"
)

type Config struct {
	Server Server
	Redis  Redis
	App    App
}

type Server struct {
	Port         int           `env:"SERVER_POST" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT" envDefault:"120s"`
}

type Redis struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
	PoolSize int    `env:"REDIS_POOL_SIZE" envDefault:"10"`

	SentinelEnabled    bool     `env:"REDIS_SENTINEL_ENABLED"     envDefault:"false"`
	SentinelMasterName string   `env:"REDIS_SENTINEL_MASTER_NAME" envDefault:"mymaster"`
	SentinelAddrs      []string `env:"REDIS_SENTINEL_ADDRS"       envSeparator:","`

	ClusterEnabled bool     `env:"REDIS_CLUSTER_ENABLED" envDefault:"false"`
	ClusterAddrs   []string `env:"REDIS_CLUSTER_ADDRS"   envSeparator:","`
}

type App struct {
	Env     string `env:"ENV" envDefault:"development"`
	Version string `env:"SERVICE_VERSION" envDefault:"dev"`
}

// ! fix this
func (cfg *Config) Validate() error {
	return nil
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("config.Load: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config.Load: validation: %w", err)
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Server: Server{
			Port:         8080,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Redis: Redis{
			Addr:     "localhost:6379",
			PoolSize: 5,
		},
		App: App{
			Env:     "test",
			Version: "test",
		},
	}
}
