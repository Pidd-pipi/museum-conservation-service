package main

import (
	"sync"
	"time"
)

type OpsTelemetryPoint struct {
	At      time.Time `json:"at"`
	Active  int       `json:"active"`
	Pending int       `json:"pending"`
}

// OpsTelemetry keeps the most recent telemetry samples of the operations
// domain. Samples are appended over time and used by the heartbeat reporter.
type OpsTelemetry struct {
	mu      sync.Mutex
	samples []OpsTelemetryPoint
	limit   int
}

func newOpsTelemetry(limit int) *OpsTelemetry {
	if limit < 1 {
		limit = 200
	}
	return &OpsTelemetry{limit: limit, samples: []OpsTelemetryPoint{}}
}

func (t *OpsTelemetry) Record(point OpsTelemetryPoint) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.samples = append(t.samples, point)
}

func (t *OpsTelemetry) Latest() (OpsTelemetryPoint, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.samples) == 0 {
		return OpsTelemetryPoint{}, false
	}
	return t.samples[len(t.samples)-1], true
}

func (t *OpsTelemetry) Series() []OpsTelemetryPoint {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.samples
}

func (t *OpsTelemetry) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.samples)
}
