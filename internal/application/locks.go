package application

import (
	"context"
	"sync"
)

type keyedLocks struct {
	mu    sync.Mutex
	locks map[string]*lockRef
}

type lockRef struct {
	ch    chan struct{}
	users int
}

func newKeyedLocks() *keyedLocks { return &keyedLocks{locks: map[string]*lockRef{}} }

// lock acquires a per-key mutex that serializes concurrent callers for the same
// key. It honors ctx so that a caller whose context is already cancelled, or
// cancels while waiting, returns promptly with ctx.Err() instead of blocking
// until the current holder releases the lock. A cancelled caller never holds
// the lock. Reference counts and lock entries are kept consistent for both
// cancelled and non-cancelled callers.
func (k *keyedLocks) lock(ctx context.Context, key string) (func(), error) {
	k.mu.Lock()
	ref := k.locks[key]
	if ref == nil {
		ref = &lockRef{ch: make(chan struct{}, 1)}
		k.locks[key] = ref
	}
	ref.users++
	k.mu.Unlock()

	select {
	case ref.ch <- struct{}{}:
		// Acquired the lock. If the context became done in the meantime,
		// release immediately so a cancelled caller never proceeds.
		if err := ctx.Err(); err != nil {
			<-ref.ch
			k.mu.Lock()
			ref.users--
			if ref.users == 0 {
				delete(k.locks, key)
			}
			k.mu.Unlock()
			return nil, err
		}
		return func() {
			<-ref.ch
			k.mu.Lock()
			ref.users--
			if ref.users == 0 {
				delete(k.locks, key)
			}
			k.mu.Unlock()
		}, nil
	case <-ctx.Done():
		k.mu.Lock()
		ref.users--
		if ref.users == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
		return nil, ctx.Err()
	}
}
