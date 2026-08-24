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

func TestBulkCompleteCanceledDoesNotHang(t *testing.T) {
	manager := newOpsSessionManager(20)
	for i := 0; i < 5; i++ {
		if _, err := manager.Open("rec-canceled-"+string(rune('a'+i)), "alice"); err != nil {
			t.Fatalf("open session: %v", err)
		}
	}
	ids := make([]string, 0, 5)
	for _, session := range manager.List() {
		ids = append(ids, session.ID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	var completed int
	var err error
	go func() {
		completed, err = manager.BulkComplete(ctx, ids)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("BulkComplete hung on canceled context (completed=%d err=%v)", completed, err)
	}
	if err == nil {
		t.Fatalf("expected cancellation error, got completed=%d err=%v", completed, err)
	}
}

func TestMetricsWindowErrorsConsistent(t *testing.T) {
	metrics := newOpsMetrics(64)
	for i := 0; i < 100; i++ {
		metrics.Record(10.0, false)
	}
	for i := 0; i < 100; i++ {
		metrics.Record(10.0, true)
	}
	view := metrics.Snapshot()
	if view.Requests != 64 {
		t.Fatalf("requests should be capped at window size, got %d", view.Requests)
	}
	if view.Errors != 64 {
		t.Fatalf("errors should reflect the window (last 64 are failures), got %d", view.Errors)
	}
	if view.Errors > view.Requests {
		t.Fatalf("errors %d exceed requests %d in a rolling window", view.Errors, view.Requests)
	}
}

func TestCompleteSessionsPartialFailureMetric(t *testing.T) {
	app := newOpsApp()
	ts := httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), app))
	defer ts.Close()

	open := func(recordID string) string {
		payload, _ := json.Marshal(map[string]string{"record_id": recordID, "owner": "alice"})
		resp, err := http.Post(ts.URL+"/api/ops/sessions", "application/json", bytes.NewReader(payload))
		if err != nil || resp.StatusCode != http.StatusCreated {
			t.Fatalf("open session status=%v err=%v", resp.StatusCode, err)
		}
		var session OpsSession
		_ = json.NewDecoder(resp.Body).Decode(&session)
		resp.Body.Close()
		return session.ID
	}

	id1 := open("rec-partial-1")
	id2 := open("rec-partial-2")
	id3 := open("rec-partial-3")

	ids := []string{id1, id2, id3, "sess-does-not-exist"}
	payload, _ := json.Marshal(map[string]any{"ids": ids})
	type result struct {
		resp *http.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := http.Post(ts.URL+"/api/ops/sessions/complete", "application/json", bytes.NewReader(payload))
		done <- result{resp: resp, err: err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("complete request: %v", r.err)
		}
		r.resp.Body.Close()
	case <-time.After(3 * time.Second):
		t.Fatalf("complete request hung")
	}

	metricsResp, err := http.Get(ts.URL + "/api/ops/metrics")
	if err != nil {
		t.Fatalf("metrics request: %v", err)
	}
	var view OpsMetricsView
	_ = json.NewDecoder(metricsResp.Body).Decode(&view)
	metricsResp.Body.Close()
	if view.Errors != 1 {
		t.Fatalf("partial completion should count as a failed batch, metrics errors=%d", view.Errors)
	}
}
