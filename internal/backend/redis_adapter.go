package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Type           string `yaml:"type"`
	Address        string `yaml:"address"`
	Password       string `yaml:"password"`
	DB             int    `yaml:"db"`
	DialTimeoutMs  int    `yaml:"dial_timeout_ms"`
	ReadTimeoutMs  int    `yaml:"read_timeout_ms"`
	WriteTimeoutMs int    `yaml:"write_timeout_ms"`
	PoolSize       int    `yaml:"pool_size"`
}

type RedisAdapter struct {
	client *redis.Client
}

func NewBackendStore(ctx context.Context, cfg Config) (BackendStore, error) {
	switch cfg.Type {
	case "redis":
		return NewRedisAdapter(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown backend store type: %s", cfg.Type)
	}
}

func NewRedisAdapter(ctx context.Context, cfg Config) (*RedisAdapter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  time.Duration(cfg.DialTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.WriteTimeoutMs) * time.Millisecond,
		PoolSize:     cfg.PoolSize,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis at constructed time: %w", err)
	}

	return &RedisAdapter{client: client}, nil
}

func (r *RedisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

func (r *RedisAdapter) Set(ctx context.Context, key string, value []byte) error {
	return r.client.Set(ctx, key, value, 0).Err()
}

func (r *RedisAdapter) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisAdapter) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisAdapter) Close() error {
	return r.client.Close()
}
