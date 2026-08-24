package main

import (
	"runtime"
	"testing"
	"time"
)

func TestHeartbeatStopStopsGoroutine(t *testing.T) {
	service := newOpsService(nil)
	telemetry := newOpsTelemetry(200)
	runtime.GC()
	before := runtime.NumGoroutine()
	for i := 0; i < 3; i++ {
		heartbeat := newOpsHeartbeat(service, telemetry)
		heartbeat.Start()
		heartbeat.Stop()
	}
	time.Sleep(150 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+1 {
		t.Fatalf("heartbeat goroutines leaked after stop: before=%d after=%d", before, after)
	}
}

func TestTelemetrySamplesBounded(t *testing.T) {
	telemetry := newOpsTelemetry(50)
	for i := 0; i < 120; i++ {
		telemetry.Record(OpsTelemetryPoint{At: time.Now().UTC()})
	}
	if telemetry.Count() != 50 {
		t.Fatalf("telemetry samples must be bounded by limit, count=%d", telemetry.Count())
	}
}

func TestTelemetrySeriesSnapshotDetached(t *testing.T) {
	telemetry := newOpsTelemetry(50)
	telemetry.Record(OpsTelemetryPoint{At: time.Now().UTC(), Active: 1})
	series := telemetry.Series()
	if len(series) != 1 {
		t.Fatalf("unexpected series: %+v", series)
	}
	series[0].Active = 999
	again := telemetry.Series()
	if again[0].Active == 999 {
		t.Fatalf("mutating a telemetry series view must not corrupt telemetry")
	}
}

func TestAuditPruneBounded(t *testing.T) {
	audit := newOpsAudit()
	for i := 0; i < 80; i++ {
		audit.Add("rec-prune", "created", "alice")
	}
	audit.Prune(20)
	if audit.Count() != 20 {
		t.Fatalf("audit must be pruned to keep limit, count=%d", audit.Count())
	}
}
