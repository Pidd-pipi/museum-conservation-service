package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func batchRecords(n int) ([]OpsRecord, []string) {
	records := make([]OpsRecord, 0, n)
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := "batch-rec-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		records = append(records, OpsRecord{
			ID: id, Subject: "batch item", Owner: "alice", Status: OpsStatusQueued, Priority: OpsPriorityNormal,
			Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
		})
		ids = append(ids, id)
	}
	return records, ids
}

func newBatchProcessor() *OpsBatchProcessor {
	store := newOpsStore(nil)
	state := newOpsStateMachine()
	manifest := newOpsManifestRegistry()
	return newOpsBatchProcessor(store, state, manifest, 8)
}

func TestBatchApplyManyRecordsFinishes(t *testing.T) {
	records, ids := batchRecords(12)
	store := newOpsStore(records)
	state := newOpsStateMachine()
	manifest := newOpsManifestRegistry()
	processor := newOpsBatchProcessor(store, state, manifest, 8)

	done := make(chan struct{})
	var result OpsBatchResult
	var err error
	go func() {
		result, err = processor.Apply(context.Background(), "alice", OpsStatusQueued, OpsStatusActive, ids)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("batch apply hung after 3s (work units never released)")
	}
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.Applied != 12 {
		t.Fatalf("expected 12 applied, got %d (skipped=%d)", result.Applied, result.Skipped)
	}
}

func TestBatchApplyReportsTransitionError(t *testing.T) {
	records, ids := batchRecords(2)
	store := newOpsStore(records)
	state := newOpsStateMachine()
	manifest := newOpsManifestRegistry()
	processor := newOpsBatchProcessor(store, state, manifest, 8)

	result, err := processor.Apply(context.Background(), "alice", OpsStatusQueued, OpsStatusPaused, ids)
	if err == nil {
		t.Fatalf("illegal transition must surface as an error, got result %+v", result)
	}
}

func TestBatchManifestReleased(t *testing.T) {
	records, ids := batchRecords(3)
	store := newOpsStore(records)
	state := newOpsStateMachine()
	manifest := newOpsManifestRegistry()
	processor := newOpsBatchProcessor(store, state, manifest, 8)

	if _, err := processor.Apply(context.Background(), "alice", OpsStatusQueued, OpsStatusActive, ids); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if manifest.Count() != 0 {
		t.Fatalf("manifest should be released after batch, active=%d", manifest.Count())
	}
}

func TestBatchHTTPReturnsResult(t *testing.T) {
	app := newOpsApp()
	records, _ := batchRecords(2)
	for _, record := range records {
		if _, err := app.Records.Create(context.Background(), record); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	ts := httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), app))
	defer ts.Close()

	payload, _ := json.Marshal(map[string]any{
		"owner": "alice", "from_status": "queued", "to_status": "active",
		"ids": []string{"batch-rec-a0", "batch-rec-b0"},
	})
	resp, err := http.Post(ts.URL+"/api/ops/batch/apply", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("batch request: %v", err)
	}
	defer resp.Body.Close()
	var result OpsBatchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("batch response must include the full result object: %v", err)
	}
	if result.BatchID == "" || result.Applied != 2 {
		t.Fatalf("batch response lost the result: %+v", result)
	}
}
