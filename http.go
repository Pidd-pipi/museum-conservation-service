package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type statusRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func NewRouter(service *ConservationService, app *OpsApp) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	if app != nil {
		router := newOpsRouter(app)
		router.register(mux)
		router.registerSessionRoutes(mux)
		router.registerBatchRoutes(mux)
	}
	mux.HandleFunc("GET /api/artifacts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.Artifacts())
	})
	mux.HandleFunc("POST /api/artifacts/status", func(w http.ResponseWriter, r *http.Request) {
		var req statusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id and status JSON are required"})
			return
		}
		item, err := service.ChangeStatus(req.ID, req.Status)
		if errors.Is(err, ErrArtifactNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, item)
	})
	return withStatic(mux)
}
