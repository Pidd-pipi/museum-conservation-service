package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// opsRouter exposes the operations domain over HTTP. Every handler is small:
// it decodes the request, calls into the app components and writes JSON.
type opsRouter struct {
	app *OpsApp
}

func newOpsRouter(app *OpsApp) *opsRouter { return &opsRouter{app: app} }

func (rt *opsRouter) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ops/records", rt.listRecords)
	mux.HandleFunc("POST /api/ops/records", rt.createRecord)
	mux.HandleFunc("GET /api/ops/records/{id}", rt.getRecord)
	mux.HandleFunc("POST /api/ops/records/{id}/transition", rt.transitionRecord)
	mux.HandleFunc("GET /api/ops/records/{id}/audit", rt.recordAudit)
	mux.HandleFunc("GET /api/ops/records/{id}/notes", rt.listNotes)
	mux.HandleFunc("POST /api/ops/records/{id}/notes", rt.addNote)
	mux.HandleFunc("GET /api/ops/snapshot", rt.snapshot)
	mux.HandleFunc("GET /api/ops/dashboard", rt.dashboard)
	mux.HandleFunc("GET /api/ops/rules", rt.rules)
	mux.HandleFunc("GET /api/ops/metrics", rt.metrics)
	mux.HandleFunc("GET /api/ops/schedule", rt.listSchedule)
	mux.HandleFunc("POST /api/ops/schedule", rt.planSchedule)
	mux.HandleFunc("POST /api/ops/schedule/{id}/move", rt.moveSchedule)
	mux.HandleFunc("GET /api/ops/alerts", rt.alerts)
	mux.HandleFunc("GET /api/ops/export", rt.export)
	mux.HandleFunc("POST /api/ops/archive/{id}", rt.archive)
}

func (rt *opsRouter) createRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string            `json:"id"`
		Subject  string            `json:"subject"`
		Owner    string            `json:"owner"`
		Priority string            `json:"priority"`
		Status   string            `json:"status"`
		Labels   map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("create", err))
		return
	}
	record := OpsRecord{
		ID:       req.ID,
		Subject:  req.Subject,
		Owner:    req.Owner,
		Priority: OpsPriority(req.Priority),
		Status:   OpsStatus(req.Status),
		Labels:   req.Labels,
	}
	if mapper := opsNewStatusMapper(); mapper != nil {
		if err := mapper.Check(record.Status); err != nil {
			opsWriteError(w, err)
			return
		}
	}
	record.Labels["source"] = "api"
	record.Labels["channel"] = "web"
	record.Labels["entered_by"] = req.Owner
	record, err := rt.app.Records.Create(r.Context(), record)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	rt.app.Metrics.Record(opsLatencyMS(time.Now()), false)
	opsJSON(w, http.StatusCreated, record)
}

func (rt *opsRouter) listRecords(w http.ResponseWriter, r *http.Request) {
	query := OpsQuery{
		Subject:  r.URL.Query().Get("subject"),
		Status:   OpsStatus(r.URL.Query().Get("status")),
		Priority: OpsPriority(r.URL.Query().Get("priority")),
		Owner:    r.URL.Query().Get("owner"),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		query.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
		query.PageSize = size
	}
	page, err := rt.app.Records.Search(r.Context(), query)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, page)
}

func (rt *opsRouter) getRecord(w http.ResponseWriter, r *http.Request) {
	record, err := rt.app.Records.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (rt *opsRouter) transitionRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Expected int    `json:"expected_revision"`
		Target   string `json:"target_status"`
		Actor    string `json:"actor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("transition", err))
		return
	}
	record, err := rt.app.Records.Transition(r.Context(), r.PathValue("id"), req.Expected, OpsStatus(req.Target), req.Actor)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (rt *opsRouter) recordAudit(w http.ResponseWriter, r *http.Request) {
	events := rt.app.Records.Audit(r.PathValue("id"))
	opsJSON(w, http.StatusOK, events)
}

func (rt *opsRouter) addNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Author string `json:"author"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("note", err))
		return
	}
	note, err := rt.app.Records.AddNote(r.PathValue("id"), req.Author, req.Text)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusCreated, note)
}

func (rt *opsRouter) listNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := rt.app.Notes.List(r.PathValue("id"))
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, notes)
}

func (rt *opsRouter) snapshot(w http.ResponseWriter, r *http.Request) {
	if snapshot, top, fresh := rt.app.Cache.Get(time.Now()); fresh {
		opsJSON(w, http.StatusOK, map[string]any{"snapshot": snapshot, "top": top, "cached": true})
		return
	}
	snapshot := rt.app.Records.Snapshot()
	opsJSON(w, http.StatusOK, map[string]any{"snapshot": snapshot, "top": []OpsRecord{}, "cached": false})
}

func (rt *opsRouter) dashboard(w http.ResponseWriter, r *http.Request) {
	view, err := rt.app.Dashboard.Build(time.Now())
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, view)
}

func (rt *opsRouter) rules(w http.ResponseWriter, r *http.Request) {
	severity := OpsPriority(r.URL.Query().Get("severity"))
	terminalOnly := r.URL.Query().Get("terminal") == "true"
	rules := opsRules()
	filtered := make([]OpsRule, 0, len(rules))
	for _, rule := range rules {
		if severity != "" && rule.Severity != severity {
			continue
		}
		if terminalOnly && !rule.Terminal {
			continue
		}
		filtered = append(filtered, rule)
	}
	opsJSON(w, http.StatusOK, filtered)
}

func (rt *opsRouter) metrics(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, rt.app.Metrics.Snapshot())
}

func (rt *opsRouter) listSchedule(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, rt.app.Schedule.List())
}

func (rt *opsRouter) planSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordID string `json:"record_id"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("schedule", err))
		return
	}
	item, err := rt.app.Schedule.Plan(req.RecordID, OpsPriority(req.Priority))
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusCreated, item)
}

func (rt *opsRouter) moveSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsWriteError(w, opsInvalid("schedule", err))
		return
	}
	item, err := rt.app.Schedule.Move(r.PathValue("id"), OpsScheduleStatus(req.Target))
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, item)
}

func (rt *opsRouter) alerts(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, rt.app.Alerts.Evaluate())
}

func (rt *opsRouter) export(w http.ResponseWriter, r *http.Request) {
	status := OpsStatus(r.URL.Query().Get("status"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	content, err := rt.app.Export.Export(r.Context(), status, limit)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (rt *opsRouter) archive(w http.ResponseWriter, r *http.Request) {
	if err := rt.app.Archive.Archive(r.Context(), r.PathValue("id")); err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, map[string]string{"archived": r.PathValue("id")})
}

func opsInvalid(operation string, cause error) error {
	return wrapOps("invalid", operation, cause)
}
