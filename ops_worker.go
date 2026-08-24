package main

import (
	"context"
	"sync/atomic"
	"time"
)

// OpsWorker is the background maintenance loop of the operations domain. It
// periodically closes stale active records, refreshes the snapshot cache and
// prunes the audit trail so in-memory state stays bounded.
type OpsWorker struct {
	service       *OpsService
	cache         *OpsSnapshotCache
	audit         *OpsAudit
	interval      time.Duration
	staleActiveTTL time.Duration
	stopCh        chan struct{}
	doneCh        chan struct{}
	started       atomic.Bool
}

func newOpsWorker(service *OpsService, cache *OpsSnapshotCache, audit *OpsAudit, interval time.Duration) *OpsWorker {
	if interval <= 0 {
		interval = time.Second
	}
	return &OpsWorker{
		service:        service,
		cache:          cache,
		audit:          audit,
		interval:       interval,
		staleActiveTTL: 7 * 24 * time.Hour,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}
}

func (w *OpsWorker) Start() {
	if !w.started.CompareAndSwap(false, true) {
		return
	}
	go w.loop()
}

func (w *OpsWorker) Stop() {
	if !w.started.CompareAndSwap(true, false) {
		return
	}
	close(w.stopCh)
	<-w.doneCh
}

func (w *OpsWorker) loop() {
	defer close(w.doneCh)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *OpsWorker) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Close stale active records before refreshing the cache so the next
	// snapshot reflects the updated statuses instead of the stalled ones.
	w.closeStaleActive(ctx, w.staleActiveTTL)
	w.refreshCache(ctx)
	w.pruneAudit(ctx, 2000)
}

func (w *OpsWorker) closeStaleActive(ctx context.Context, maxAge time.Duration) {
	items, err := w.service.store.List(ctx)
	if err != nil {
		return
	}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return
		}
		if item.Status != OpsStatusActive {
			continue
		}
		updated, err := opsParseStamp(item.UpdatedAt)
		if err != nil || time.Since(updated) < maxAge {
			continue
		}
		_, _ = w.service.Transition(ctx, item.ID, item.Revision, OpsStatusClosed, "worker")
	}
}

func (w *OpsWorker) refreshCache(ctx context.Context) {
	items, err := w.service.store.List(ctx)
	if err != nil {
		return
	}
	w.cache.Refresh(items, time.Now())
}

func (w *OpsWorker) pruneAudit(ctx context.Context, keep int) {
	w.audit.Prune(keep)
}
