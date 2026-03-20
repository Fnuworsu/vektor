package backend

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func getTestConfig() Config {
	return Config{
		Type:           "redis",
		Address:        "localhost:6379",
		Password:       "",
		DB:             0,
		DialTimeoutMs:  500,
		ReadTimeoutMs:  100,
		WriteTimeoutMs: 100,
		PoolSize:       10,
	}
}

func TestGetMiss(t *testing.T) {
	ctx := context.Background()
	store, err := NewBackendStore(ctx, getTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	val, err := store.Get(ctx, "nonexistent_key")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if val != nil {
		t.Fatalf("expected nil value, got: %v", val)
	}
}

func TestSetAndGet(t *testing.T) {
	ctx := context.Background()
	store, err := NewBackendStore(ctx, getTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	key := "test_key"
	expected := []byte("test_value")

	err = store.Set(ctx, key, expected)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	val, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if !bytes.Equal(val, expected) {
		t.Fatalf("expected %s, got %s", expected, string(val))
	}
}

func TestTTLExpiry(t *testing.T) {
	ctx := context.Background()
	cfg := getTestConfig()
	store, err := NewBackendStore(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	rawClient := redis.NewClient(&redis.Options{Addr: cfg.Address})
	defer rawClient.Close()

	key := "ttl_key"
	err = rawClient.Set(ctx, key, []byte("ttl_value"), 50*time.Millisecond).Err()
	if err != nil {
		t.Fatalf("failed to raw set: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	val, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("expected no error after expiry, got: %v", err)
	}
	if val != nil {
		t.Fatalf("expected nil value after expiry, got: %v", val)
	}
}

func TestPing(t *testing.T) {
	ctx := context.Background()
	store, err := NewBackendStore(ctx, getTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.Ping(ctx)
	if err != nil {
		t.Fatalf("expected no error on ping, got: %v", err)
	}
}

func TestClose(t *testing.T) {
	ctx := context.Background()
	store, err := NewBackendStore(ctx, getTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got: %v", err)
	}

	err = store.Ping(ctx)
	if err == nil {
		t.Fatalf("expected error on ping after close, got nil")
	}
}
