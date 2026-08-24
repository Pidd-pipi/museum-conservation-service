package main

import (
	"context"
	"testing"
	"time"
)

func TestStoreContextNotReused(t *testing.T) {
	store := newOpsStore([]OpsRecord{{
		ID: "ctx-rec-1", Subject: "inspect", Owner: "alice", Status: OpsStatusActive, Priority: OpsPriorityNormal,
		Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
	}})
	// First request with a short deadline that expires.
	shortCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := store.Get(shortCtx, "ctx-rec-1"); err != nil {
		t.Fatalf("first get: %v", err)
	}
	time.Sleep(80 * time.Millisecond)

	// A fresh request must not inherit the expired deadline.
	record, err := store.Get(context.Background(), "ctx-rec-1")
	if err != nil {
		t.Fatalf("fresh request must not reuse stale context: %v", err)
	}
	if record.ID != "ctx-rec-1" {
		t.Fatalf("wrong record: %v", record.ID)
	}
}

func TestStoreListContextNotReused(t *testing.T) {
	store := newOpsStore([]OpsRecord{{
		ID: "ctx-list-1", Subject: "inspect", Owner: "alice", Status: OpsStatusActive, Priority: OpsPriorityNormal,
		Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
	}})
	shortCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := store.List(shortCtx); err != nil {
		t.Fatalf("first list: %v", err)
	}
	time.Sleep(80 * time.Millisecond)

	items, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("fresh list must not reuse stale context: %v", err)
	}
	if len(items) != 1 || items[0].ID != "ctx-list-1" {
		t.Fatalf("wrong list result: %+v", items)
	}
}

func TestExportStopsOnCancel(t *testing.T) {
	records := make([]OpsRecord, 0, 50)
	for i := 0; i < 50; i++ {
		records = append(records, OpsRecord{
			ID: "exp-" + string(rune('a'+i%26)), Subject: "item", Owner: "alice", Status: OpsStatusQueued,
			Priority: OpsPriorityNormal, Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
		})
	}
	store := newOpsStore(records)
	exportService := newOpsExportService(store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := exportService.Export(ctx, "", 500)
	if err == nil {
		t.Fatalf("export must stop when context is canceled")
	}
}

func TestArchiveHonorsCancel(t *testing.T) {
	store := newOpsStore([]OpsRecord{{
		ID: "arch-ctx-1", Subject: "closed item", Owner: "alice", Status: OpsStatusClosed, Priority: OpsPriorityLow,
		Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
	}})
	archive := newOpsArchive(store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := archive.Archive(ctx, "arch-ctx-1"); err == nil {
		t.Fatalf("archive must honor context cancellation")
	}
	if _, err := archive.Lookup(context.Background(), "arch-ctx-1"); err != nil {
		t.Fatalf("record must not be archived when context is canceled")
	}
}
