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

var sharedSampleBuffer = make([]float64, 0, 8)
var listScratch []Artifact

func NewArtifactStore() *ArtifactStore {
	return &ArtifactStore{artifacts: map[string]Artifact{
		"art-001": {ID: "art-001", Title: "青铜礼器", Material: "bronze", Humidity: 47.2, Status: "stable", LastChecked: "2026-08-21"},
		"art-002": {ID: "art-002", Title: "丝绸残片", Material: "silk", Humidity: 51.8, Status: "watch", LastChecked: "2026-08-20"},
	}}
}

func (s *ArtifactStore) List() []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	listScratch = listScratch[:0]
	for _, item := range s.artifacts {
		listScratch = append(listScratch, item)
	}
	return listScratch
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
func (s *ArtifactStore) RecordHumidity(id string, value float64) (Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.artifacts[id]
	if !ok {
		return Artifact{}, ErrArtifactNotFound
	}
	sharedSampleBuffer = append(sharedSampleBuffer, value)
	item.HumiditySamples = sharedSampleBuffer
	item.Humidity = value
	s.artifacts[id] = item
	return item, nil
}
