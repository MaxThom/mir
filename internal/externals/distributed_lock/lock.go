package distributed_lock

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Memory vs File Storage
// lock was acquired
// nats crashes and restarts
// if memory
//   cannot acquire nor releases
//   when back, there is no stuck lock
//   another process can get the same lock and double execution
//   if trying to release, it will be ok
// if file
//   cannot acquire nor releases
//   when back, locks are still there
//   another process cant get the same lock
//   but if trying to release while down, we got a forever lock
// main diff is, memory can lead to double execution, while file forever lock
// solution could be file with ttl
// version 2.11 will have ttl per keys. planned release in 2024

type DistributedLockStore interface {
	TryAcquire(key string, ttl time.Duration) (bool, error)
	Acquire(ctx context.Context, key string, ttl time.Duration, timeout time.Duration) error
	Release(key string) error
}

type natsLockStore struct {
	kv    nats.KeyValue
	owner string
}

func NewNatsLockStore(bus *nats.Conn, owner string) (DistributedLockStore, error) {
	js, err := bus.JetStream()
	if err != nil {
		return nil, err
	}
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:      "__mir_locks",
		Description: "distributed locking mechanism for Mir",
		Storage:     nats.FileStorage,
		TTL:         5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return &natsLockStore{
		kv:    kv,
		owner: owner,
	}, nil
}

// TryAcquire will return immedietly if the lock is not free.
// TTL is the maximum time the lock will be held.
// ** TTL will be built in NatsIO 2.11 **
func (s *natsLockStore) TryAcquire(key string, ttl time.Duration) (bool, error) {
	entry, err := s.kv.Get(key)
	if err != nil && err != nats.ErrKeyNotFound {
		return false, fmt.Errorf("%w", err)
	}

	// If key exists, check if it's expired
	if entry != nil {
		if time.Since(entry.Created()) > ttl {
			// Expired
			if err := s.kv.Purge(key); err != nil {
				return false, fmt.Errorf("failed to delete expired lock: %w", err)
			}
		} else {
			return false, nil
		}
	}

	// Try to acquire the lock
	if _, err = s.kv.Create(key, []byte(s.owner)); err != nil {
		if err == nats.ErrKeyExists {
			return false, nil
		}
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return true, nil
}

// Acquire will block until the lock is acquired or an error occurs.
// Timeout is the maximum time to wait for the lock.
func (s *natsLockStore) Acquire(ctx context.Context, key string, ttl time.Duration, timeout time.Duration) error {
	const (
		initialBackoff = 100 * time.Millisecond
		maxBackoff     = time.Second
	)
	backoff := initialBackoff

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		acquired, err := s.TryAcquire(key, ttl)
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}

		if acquired {
			return nil
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("timeout waiting for lock: %w", ctx.Err())
		case <-timer.C:
		}

		// Exponential backoff with max limit
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (s *natsLockStore) Release(key string) error {
	if err := s.kv.Purge(key); err != nil && err != nats.ErrKeyNotFound {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}
