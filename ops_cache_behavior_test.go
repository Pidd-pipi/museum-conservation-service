package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func seedCacheRecords(n int) []OpsRecord {
	records := make([]OpsRecord, 0, n)
	for i := 0; i < n; i++ {
		records = append(records, OpsRecord{
			ID:        fmt.Sprintf("r-%02d", i),
			Subject:   "conservation item",
			Owner:     "alice",
			Status:    OpsStatusActive,
			Priority:  OpsPriorityNormal,
			Labels:    map[string]string{"site": "hall-a"},
			UpdatedAt: timeNowOps(),
		})
	}
	return records
}

func TestSnapshotCacheConcurrentAccess(t *testing.T) {
	cache := newOpsSnapshotCache(time.Minute)
	records := seedCacheRecords(40)
	cache.Refresh(records, time.Now())

	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 400; i++ {
				_, _, _ = cache.Get(time.Now())
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			cache.Refresh(records, time.Now())
		}
	}()
	close(start)
	wg.Wait()
}

func TestSnapshotCacheViewFrozenAfterRefresh(t *testing.T) {
	cache := newOpsSnapshotCache(time.Minute)
	first := []OpsRecord{{
		ID: "a", Subject: "alpha", Owner: "alice", Status: OpsStatusActive, Priority: OpsPriorityHigh,
		Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
	}}
	cache.Refresh(first, time.Now())
	snapshot, top, ok := cache.Get(time.Now())
	if !ok || len(top) != 1 || top[0].ID != "a" {
		t.Fatalf("expected top record a, ok=%v top=%v", ok, top)
	}
	if snapshot.ByStatus[OpsStatusActive] != 1 {
		t.Fatalf("snapshot should count active=1, got %+v", snapshot.ByStatus)
	}
	second := []OpsRecord{{
		ID: "b", Subject: "beta", Owner: "bob", Status: OpsStatusClosed, Priority: OpsPriorityLow,
		Labels: map[string]string{"site": "hall-b"}, UpdatedAt: timeNowOps(),
	}}
	cache.Refresh(second, time.Now())
	if len(top) != 1 || top[0].ID != "a" {
		t.Fatalf("old snapshot view was mutated by later refresh: %+v", top)
	}
	_, top2, _ := cache.Get(time.Now())
	if len(top2) != 1 || top2[0].ID != "b" {
		t.Fatalf("new view should contain b: %+v", top2)
	}
}

func TestWorkerTickSnapshotFreshAfterClose(t *testing.T) {
	store := newOpsStore([]OpsRecord{{
		ID: "stale-1", Subject: "old active", Owner: "alice", Status: OpsStatusActive, Priority: OpsPriorityNormal,
		Labels: map[string]string{"site": "hall-a"}, UpdatedAt: time.Now().Add(-40 * 24 * time.Hour).Format(time.RFC3339Nano),
	}})
	audit := newOpsAudit()
	state := newOpsStateMachine()
	clock := newOpsClock()
	notes := newOpsNoteStore(store)
	service := newOpsServiceWith(store, audit, state, clock, notes)
	cache := newOpsSnapshotCache(time.Minute)
	worker := newOpsWorker(service, cache, audit, time.Hour)
	worker.tick()
	snapshot, _, ok := cache.Get(time.Now())
	if !ok {
		t.Fatalf("cache should be fresh after tick")
	}
	if snapshot.Active != 0 {
		t.Fatalf("stale active record should have been closed before snapshot refresh, active=%d records=%d", snapshot.Active, snapshot.Records)
	}
}
