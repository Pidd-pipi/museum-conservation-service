package main

import "testing"

func TestHumiditySamplesCrossArtifactIsolated(t *testing.T) {
	store := NewArtifactStore()
	if _, err := store.RecordHumidity("art-001", 47.5); err != nil {
		t.Fatalf("record art-001: %v", err)
	}
	if _, err := store.RecordHumidity("art-002", 51.0); err != nil {
		t.Fatalf("record art-002: %v", err)
	}
	items := store.List()
	var first *Artifact
	for i := range items {
		if items[i].ID == "art-001" {
			first = &items[i]
			break
		}
	}
	if first == nil {
		t.Fatalf("art-001 missing from list")
	}
	if len(first.HumiditySamples) != 1 || first.HumiditySamples[0] != 47.5 {
		t.Fatalf("art-001 samples must not include art-002 readings: %+v", first.HumiditySamples)
	}
	for i := range items {
		if items[i].ID == "art-002" {
			if len(items[i].HumiditySamples) != 1 || items[i].HumiditySamples[0] != 51.0 {
				t.Fatalf("art-002 samples must not include art-001 readings: %+v", items[i].HumiditySamples)
			}
			break
		}
	}
}

func TestListSamplesDoNotAliasStore(t *testing.T) {
	store := NewArtifactStore()
	if _, err := store.RecordHumidity("art-001", 47.5); err != nil {
		t.Fatalf("record: %v", err)
	}
	first := store.List()
	var target *Artifact
	for i := range first {
		if first[i].ID == "art-001" {
			target = &first[i]
			break
		}
	}
	target.HumiditySamples[0] = 999

	again := store.List()
	for i := range again {
		if again[i].ID == "art-001" {
			if again[i].HumiditySamples[0] == 999 {
				t.Fatalf("mutating a list sample must not corrupt the store")
			}
			break
		}
	}
}

func TestHumidityTrendSnapshotStable(t *testing.T) {
	service := NewConservationService(NewArtifactStore())
	if _, err := service.store.RecordHumidity("art-001", 47.5); err != nil {
		t.Fatalf("record: %v", err)
	}
	first := service.HumidityTrend()
	if len(first) != 1 || first[0].Latest != 47.5 {
		t.Fatalf("unexpected first trend: %+v", first)
	}
	if _, err := service.store.RecordHumidity("art-002", 51.0); err != nil {
		t.Fatalf("record 2: %v", err)
	}
	second := service.HumidityTrend()
	second[0].Latest = 999
	if first[0].Latest == 999 {
		t.Fatalf("mutating a later trend view must not corrupt an earlier view: %+v", first)
	}
}

func TestListCallsKeepOwnBuffer(t *testing.T) {
	store := NewArtifactStore()
	first := store.List()
	if len(first) == 0 {
		t.Fatalf("expected seed artifacts")
	}
	first[0].Humidity = 123
	if _, err := store.RecordHumidity("art-001", 47.5); err != nil {
		t.Fatalf("record: %v", err)
	}
	_ = store.List()
	if first[0].Humidity == 123 {
		return
	}
	if first[0].Humidity != 123 {
		t.Fatalf("previously returned list was mutated by a later list call: %+v", first)
	}
}
