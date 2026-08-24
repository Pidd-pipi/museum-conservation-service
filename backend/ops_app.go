package main

import "time"

// OpsApp wires together all operations-domain components and is the single
// access point for HTTP handlers and the background worker.
type OpsApp struct {
	Records   *OpsService
	Store     *OpsStore
	Audit     *OpsAudit
	Notes     *OpsNoteStore
	Sessions  *OpsSessionManager
	Metrics   *OpsMetrics
	Search    *OpsSearchIndex
	Cache     *OpsSnapshotCache
	Batch     *OpsBatchProcessor
	Schedule  *OpsScheduleService
	Alerts    *OpsAlertEngine
	Heartbeat *OpsHeartbeat
	Telemetry *OpsTelemetry
	Export    *OpsExportService
	Archive   *OpsArchive
	Manifest  *OpsManifestRegistry
	Worker    *OpsWorker
	Dashboard *OpsDashboard
}

func newOpsApp() *OpsApp {
	store := newOpsStore(nil)
	audit := newOpsAudit()
	state := newOpsStateMachine()
	clock := newOpsClock()
	notes := newOpsNoteStore(store)
	service := newOpsServiceWith(store, audit, state, clock, notes)
	manifest := newOpsManifestRegistry()
	cache := newOpsSnapshotCache(5 * time.Second)
	search := newOpsSearchIndex()
	sessions := newOpsSessionManager(50)
	metrics := newOpsMetrics(256)
	telemetry := newOpsTelemetry(200)
	schedule := newOpsScheduleService()
	alerts := newOpsAlertEngine(schedule)
	exportService := newOpsExportService(store)
	archive := newOpsArchive(store)
	batch := newOpsBatchProcessor(store, state, manifest, 8)
	worker := newOpsWorker(service, cache, audit, 2*time.Second)
	heartbeat := newOpsHeartbeat(service, telemetry)
	dashboard := newOpsDashboard(cache, search)
	return &OpsApp{
		Records:   service,
		Store:     store,
		Audit:     audit,
		Notes:     notes,
		Sessions:  sessions,
		Metrics:   metrics,
		Search:    search,
		Cache:     cache,
		Batch:     batch,
		Schedule:  schedule,
		Alerts:    alerts,
		Heartbeat: heartbeat,
		Telemetry: telemetry,
		Export:    exportService,
		Archive:   archive,
		Manifest:  manifest,
		Worker:    worker,
		Dashboard: dashboard,
	}
}
