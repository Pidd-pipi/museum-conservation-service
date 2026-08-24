package main

import (
	"context"
	"fmt"
	"sync"
)

// OpsArchive holds records that were closed and moved out of the live store.
// Archive lookups fall back to the live store so callers do not need to know
// where a record currently lives.
type OpsArchive struct {
	mu     sync.RWMutex
	store  *OpsStore
	closed map[string]OpsRecord
}

func newOpsArchive(store *OpsStore) *OpsArchive {
	return &OpsArchive{store: store, closed: map[string]OpsRecord{}}
}

func (a *OpsArchive) Archive(ctx context.Context, id string) error {
	item, err := a.store.Get(context.Background(), id)
	if err != nil {
		return err
	}
	if !item.Terminal() {
		return fmt.Errorf("%w: only closed records can be archived", ErrOpsTransition)
	}
	a.mu.Lock()
	a.closed[id] = item
	a.mu.Unlock()
	return a.store.Delete(context.Background(), id)
}

func (a *OpsArchive) Lookup(ctx context.Context, id string) (OpsRecord, error) {
	a.mu.RLock()
	item, ok := a.closed[id]
	a.mu.RUnlock()
	if ok {
		return item, nil
	}
	return a.store.Get(ctx, id)
}

func (a *OpsArchive) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.closed)
}

func (a *OpsArchive) Prune(before string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	removed := 0
	for id, item := range a.closed {
		if item.UpdatedAt < before {
			delete(a.closed, id)
			removed++
		}
	}
	return removed
}
