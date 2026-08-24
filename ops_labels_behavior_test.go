package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRecordWithLabelsSucceeds(t *testing.T) {
	app := newOpsApp()
	ts := httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), app))
	defer ts.Close()

	payload, _ := json.Marshal(map[string]any{
		"id": "op-labels-1", "subject": "除锈", "owner": "alice", "priority": "high", "status": "active",
		"labels": map[string]string{"site": "hall-a"},
	})
	resp, err := http.Post(ts.URL+"/api/ops/records", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create with labels should succeed, got %d", resp.StatusCode)
	}
	var record OpsRecord
	_ = json.NewDecoder(resp.Body).Decode(&record)
	if record.Labels["site"] != "hall-a" {
		t.Fatalf("site label missing: %+v", record.Labels)
	}
	if record.Labels["source"] != "api" {
		t.Fatalf("source label missing: %+v", record.Labels)
	}
}

func TestCreateRecordWithoutLabelsValidated(t *testing.T) {
	app := newOpsApp()
	ts := httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), app))
	defer ts.Close()

	payload, _ := json.Marshal(map[string]any{
		"id": "op-nolabels-1", "subject": "除尘", "owner": "bob", "priority": "normal",
	})
	resp, err := http.Post(ts.URL+"/api/ops/records", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create without required site label should be rejected with 400, got %d", resp.StatusCode)
	}
}

func TestCloneRecordLabelsIsolated(t *testing.T) {
	original := OpsRecord{
		ID: "rec-clone-1", Subject: "inspect", Owner: "alice", Priority: OpsPriorityNormal,
		Labels: map[string]string{"site": "hall-a"},
	}
	cloned := original.Clone()
	cloned.Labels["site"] = "hall-b"
	cloned.Labels["extra"] = "x"
	if original.Labels["site"] != "hall-a" {
		t.Fatalf("clone must not mutate original labels: %+v", original.Labels)
	}
	if _, ok := original.Labels["extra"]; ok {
		t.Fatalf("clone must not add labels to original: %+v", original.Labels)
	}
}
