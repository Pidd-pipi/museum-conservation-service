package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func (rt *opsRouter) registerSessionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/ops/sessions", rt.openSession)
	mux.HandleFunc("POST /api/ops/sessions/complete", rt.completeSessions)
	mux.HandleFunc("GET /api/ops/sessions", rt.listSessions)
}

func (rt *opsRouter) openSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordID string `json:"record_id"`
		Owner    string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("session", err))
		return
	}
	session, err := rt.app.Sessions.Open(req.RecordID, req.Owner)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	rt.app.Metrics.Record(opsLatencyMS(time.Now()), false)
	opsJSON(w, http.StatusCreated, session)
}

func (rt *opsRouter) completeSessions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("session", err))
		return
	}
	completed, err := rt.app.Sessions.BulkComplete(r.Context(), req.IDs)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	rt.app.Metrics.Record(opsLatencyMS(time.Now()), completed == 0)
	opsJSON(w, http.StatusOK, map[string]int{"completed": completed})
}

func (rt *opsRouter) listSessions(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, rt.app.Sessions.List())
}
