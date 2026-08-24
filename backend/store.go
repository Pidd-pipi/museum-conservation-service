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

func (s *ArtifactStore) List() []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Artifact, 0, len(s.artifacts))
	for _, item := range s.artifacts {
		items = append(items, item)
	}
	return items
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
