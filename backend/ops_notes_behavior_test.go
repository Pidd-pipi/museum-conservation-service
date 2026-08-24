package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoteStoreNotFoundChainPreserved(t *testing.T) {
	store := newOpsStore(nil)
	notes := newOpsNoteStore(store)
	_, err := notes.Add("missing-record", "alice", "inspect")
	if err == nil {
		t.Fatalf("expected error for missing record")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("error chain must preserve ErrOpsNotFound, got: %v", err)
	}
}

func TestAddNoteServiceNotFoundPreserved(t *testing.T) {
	service := newOpsService(nil)
	_, err := service.AddNote("missing-record", "alice", "inspect")
	if err == nil {
		t.Fatalf("expected error for missing record")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("service must surface ErrOpsNotFound, got: %v", err)
	}
}

func TestAddNoteHTTPStatusCodes(t *testing.T) {
	app := newOpsApp()
	ts := httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), app))
	defer ts.Close()

	// Missing record -> 404
	payload, _ := json.Marshal(map[string]string{"author": "alice", "text": "check"})
	resp, err := http.Post(ts.URL+"/api/ops/records/no-such/notes", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("note request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing record note should be 404, got %d", resp.StatusCode)
	}

	// Empty author -> 400 (record exists)
	create, _ := json.Marshal(map[string]any{"id": "op-note-1", "subject": "conservation", "owner": "alice", "priority": "normal", "labels": map[string]string{"site": "hall-a"}})
	resp, err = http.Post(ts.URL+"/api/ops/records", "application/json", bytes.NewReader(create))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create record status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
	payload, _ = json.Marshal(map[string]string{"author": "", "text": "check"})
	resp, err = http.Post(ts.URL+"/api/ops/records/op-note-1/notes", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("note request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty author note should be 400, got %d", resp.StatusCode)
	}
}
