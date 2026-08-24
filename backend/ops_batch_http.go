package main

import (
	"encoding/json"
	"net/http"
)

func (rt *opsRouter) registerBatchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/ops/batch/apply", rt.applyBatch)
	mux.HandleFunc("GET /api/ops/manifests", rt.listManifests)
}

func (rt *opsRouter) applyBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner string   `json:"owner"`
		From  string   `json:"from_status"`
		To    string   `json:"to_status"`
		IDs   []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("batch", err))
		return
	}
	_, err := rt.app.Batch.Apply(r.Context(), req.Owner, OpsStatus(req.From), OpsStatus(req.To), req.IDs)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (rt *opsRouter) listManifests(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, map[string]int{"active": rt.app.Manifest.Count()})
}
