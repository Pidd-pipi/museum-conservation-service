package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer() *httptest.Server {
	return httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), newOpsApp()))
}

func TestHTTPWorkflow(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
	resp, err = http.Get(ts.URL + "/api/artifacts")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
	payload, _ := json.Marshal(map[string]string{"id": "art-001", "status": "treatment"})
	resp, err = http.Post(ts.URL+"/api/artifacts/status", "application/json", bytes.NewReader(payload))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
	resp, err = http.Post(ts.URL+"/api/artifacts/status", "application/json", bytes.NewReader([]byte(`{"id":"art-001","status":"lost"}`)))
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
	resp, err = http.Get(ts.URL + "/")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("index status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
}
