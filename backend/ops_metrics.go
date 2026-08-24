package main

import (
	"sort"
	"sync"
	"time"
)

type OpsMetricsView struct {
	Requests   int     `json:"requests"`
	Errors     int     `json:"errors"`
	AvgLatency float64 `json:"avg_latency_ms"`
	P95Latency float64 `json:"p95_latency_ms"`
}

// OpsMetrics keeps a rolling window of recent request latencies and counts.
// The window is bounded so old samples are dropped instead of growing forever.
type OpsMetrics struct {
	mu       sync.Mutex
	capacity int
	latency  []float64
	errors   int
}

func newOpsMetrics(capacity int) *OpsMetrics {
	if capacity < 1 {
		capacity = 64
	}
	return &OpsMetrics{capacity: capacity, latency: []float64{}}
}

func (m *OpsMetrics) Record(latency float64, failed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = append(m.latency, latency)
	if len(m.latency) > m.capacity {
		m.latency = m.latency[len(m.latency)-m.capacity:]
	}
	if failed {
		m.errors++
	}
}

func (m *OpsMetrics) Snapshot() OpsMetricsView {
	m.mu.Lock()
	defer m.mu.Unlock()
	view := OpsMetricsView{Requests: len(m.latency), Errors: m.errors}
	if len(m.latency) > 0 {
		total := 0.0
		samples := make([]float64, len(m.latency))
		copy(samples, m.latency)
		sort.Float64s(samples)
		for _, value := range samples {
			total += value
		}
		view.AvgLatency = total / float64(len(samples))
		view.P95Latency = samples[(len(samples)*95)/100]
	}
	return view
}

func (m *OpsMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = m.latency[:0]
	m.errors = 0
}

func opsLatencyMS(start time.Time) float64 { return float64(time.Since(start).Microseconds()) / 1000.0 }
