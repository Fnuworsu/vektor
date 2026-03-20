package proxy

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/events"
	"github.com/redis/go-redis/v9"
)

func TestProxyConcurrencyAndEvents(t *testing.T) {
	ctx := context.Background()
	cfg := backend.Config{
		Type:           "redis",
		Address:        "localhost:6379",
		Password:       "",
		DB:             0,
		DialTimeoutMs:  500,
		ReadTimeoutMs:  100,
		WriteTimeoutMs: 100,
		PoolSize:       50,
	}

	store, err := backend.NewBackendStore(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		t.Skipf("redis not available: %v", err)
	}

	eventCh := make(chan events.AccessEvent, 1000)
	server := NewServer("localhost:6380", store, eventCh)
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6380"})
	defer client.Close()

	err = client.Set(ctx, "concc_key", "val", 0).Err()
	if err != nil {
		t.Fatalf("failed setup set: %v", err)
	}

	var wg sync.WaitGroup
	workers := 50
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				val, err := client.Get(ctx, "concc_key").Result()
				if err != nil {
					t.Errorf("worker %d get err: %v", id, err)
				}
				if val != "val" {
					t.Errorf("worker %d val mismatch: %s", id, val)
				}
			}
		}(i)
	}
	wg.Wait()

	eventsCount := 0
	timeout := time.After(2 * time.Second)
eventLoop:
	for {
		select {
		case ev := <-eventCh:
			if ev.Key == "concc_key" {
				eventsCount++
			}
			if eventsCount == workers*10 {
				break eventLoop
			}
		case <-timeout:
			break eventLoop
		}
	}

	if eventsCount != workers*10 {
		t.Fatalf("expected %d events, got %d", workers*10, eventsCount)
	}
}

func TestProxyUnsupportedCommand(t *testing.T) {
	ctx := context.Background()
	cfg := backend.Config{
		Type:           "redis",
		Address:        "localhost:6379",
		Password:       "",
		DB:             0,
		DialTimeoutMs:  500,
		ReadTimeoutMs:  100,
		WriteTimeoutMs: 100,
		PoolSize:       10,
	}

	store, err := backend.NewBackendStore(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		t.Skipf("redis not available: %v", err)
	}

	eventCh := make(chan events.AccessEvent, 100)
	server := NewServer("localhost:6381", store, eventCh)
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6381"})
	defer client.Close()

	err = client.Do(ctx, "HGETALL", "foo").Err()
	if err == nil || !strings.Contains(err.Error(), "COMMAND_NOT_SUPPORTED") {
		t.Fatalf("expected COMMAND_NOT_SUPPORTED, got: %v", err)
	}
}
