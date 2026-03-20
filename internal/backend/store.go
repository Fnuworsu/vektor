package backend

import (
	"context"
)

type BackendStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close() error
}