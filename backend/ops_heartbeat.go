package main

import (
	"context"
	"sync"
	"time"
)

// OpsHeartbeat periodically samples the operations domain and pushes the
// sample into telemetry. It is started by the runtime and must stop cleanly
// when the service shuts down.
type OpsHeartbeat struct {
	service   *OpsService
	telemetry *OpsTelemetry
	interval  time.Duration
	stopCh    chan struct{}
	doneCh    chan struct{}
	mu        sync.Mutex
	running   bool
}

func newOpsHeartbeat(service *OpsService, telemetry *OpsTelemetry) *OpsHeartbeat {
	return &OpsHeartbeat{service: service, telemetry: telemetry, interval: 50 * time.Millisecond, stopCh: make(chan struct{}), doneCh: make(chan struct{})}
}

func (h *OpsHeartbeat) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()
	go h.loop()
}

func (h *OpsHeartbeat) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()
}

func (h *OpsHeartbeat) loop() {
	defer close(h.doneCh)
	ctx := context.Background()
	for {
		h.sample()
		timer := time.NewTimer(h.interval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (h *OpsHeartbeat) sample() {
	snapshot := h.service.Snapshot()
	h.telemetry.Record(OpsTelemetryPoint{At: nowOpsTime(), Active: snapshot.Active, Pending: snapshot.Records - snapshot.Active})
}

func nowOpsTime() time.Time { return time.Now().UTC() }
