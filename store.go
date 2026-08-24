package main

import (
	"errors"
	"sync"
)

var ErrArtifactNotFound = errors.New("artifact not found")

type ArtifactStore struct {
	mu        sync.RWMutex
	artifacts map[string]Artifact
}

func NewArtifactStore() *ArtifactStore {
	return &ArtifactStore{artifacts: map[string]Artifact{
		"art-001": {ID: "art-001", Title: "青铜礼器", Material: "bronze", Humidity: 47.2, Status: "stable", LastChecked: "2026-08-21"},
		"art-002": {ID: "art-002", Title: "丝绸残片", Material: "silk", Humidity: 51.8, Status: "watch", LastChecked: "2026-08-20"},
	}}
}

// List returns a fresh, independently owned snapshot of every artifact.
// Each returned Artifact (including its HumiditySamples slice) is copied so
// that callers cannot mutate the store through the returned value, and
// repeated calls do not alias one another's backing arrays.
func (s *ArtifactStore) List() []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Artifact, 0, len(s.artifacts))
	for _, item := range s.artifacts {
		item.HumiditySamples = cloneSamples(item.HumiditySamples)
		out = append(out, item)
	}
	return out
}

func (s *ArtifactStore) UpdateStatus(id, status string) (Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.artifacts[id]
	if !ok {
		return Artifact{}, ErrArtifactNotFound
	}
	item.Status = status
	s.artifacts[id] = item
	return item, nil
}

// RecordHumidity records a new humidity reading for an artifact.
// The reading is appended to that artifact's own samples slice, isolated
// from every other artifact and from the value returned to the caller.
func (s *ArtifactStore) RecordHumidity(id string, value float64) (Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.artifacts[id]
	if !ok {
		return Artifact{}, ErrArtifactNotFound
	}
	item.HumiditySamples = append(item.HumiditySamples, value)
	item.Humidity = value
	s.artifacts[id] = item
	item.HumiditySamples = cloneSamples(item.HumiditySamples)
	return item, nil
}

// cloneSamples returns a copy of in with its own backing array, preserving
// a nil input as nil so the JSON encoding (null vs []) is unchanged.
func cloneSamples(in []float64) []float64 {
	if in == nil {
		return nil
	}
	out := make([]float64, len(in))
	copy(out, in)
	return out
}
