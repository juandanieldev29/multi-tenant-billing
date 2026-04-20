package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

type IdempotencyStore struct {
	client *redis.Client
}

func NewIdempotencyStore(addr string) *IdempotencyStore {
	return &IdempotencyStore{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

// Acquire returns true if the key was set (first time seen), false if duplicate.
func (s *IdempotencyStore) Acquire(ctx context.Context, tenantID, eventID string) (bool, error) {
	key := fmt.Sprintf("idempotency:%s:%s", tenantID, eventID)
	return s.client.SetNX(ctx, key, 1, idempotencyTTL).Result()
}
