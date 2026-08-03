// Package registry provides a Redis-backed service registry with
// TTL-based heartbeats: instances register themselves and refresh
// periodically; a crashed instance's entry simply expires on its own
// after a few missed heartbeats, with no explicit failure detection
// needed anywhere else in the system.
package registry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "nodefall:instances:"

// Instance describes one registered service instance.
type Instance struct {
	ID       string `json:"id"`
	GRPCAddr string `json:"grpc_addr"`
}

// Registry wraps a Redis client for registering and discovering
// instances.
type Registry struct {
	client *redis.Client
}

// New creates a Registry connected to the given Redis address
// (host:port, no scheme).
func New(addr string) *Registry {
	return &Registry{client: redis.NewClient(&redis.Options{Addr: addr})}
}

// Register writes inst to Redis with the given TTL. Call this on
// startup and repeatedly via Heartbeat — an entry that stops being
// refreshed simply expires.
func (r *Registry) Register(ctx context.Context, inst Instance, ttl time.Duration) error {
	data, err := json.Marshal(inst)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, keyPrefix+inst.ID, data, ttl).Err()
}

// Heartbeat registers inst immediately, then re-registers every
// interval until ctx is cancelled. Intended to run on its own
// goroutine for the lifetime of the instance's process.
func (r *Registry) Heartbeat(ctx context.Context, inst Instance, ttl, interval time.Duration) {
	r.Register(ctx, inst, ttl)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Register(ctx, inst, ttl)
		}
	}
}

// ListLive returns every currently-registered (non-expired) instance.
func (r *Registry) ListLive(ctx context.Context) ([]Instance, error) {
	var instances []Instance

	iter := r.client.Scan(ctx, 0, keyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		data, err := r.client.Get(ctx, iter.Val()).Result()
		if err != nil {
			continue // expired between Scan and Get — skip, not an error
		}
		var inst Instance
		if err := json.Unmarshal([]byte(data), &inst); err != nil {
			continue
		}
		instances = append(instances, inst)
	}
	return instances, iter.Err()
}
