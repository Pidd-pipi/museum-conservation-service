package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type OpsScheduleStatus string

const (
	OpsSchedulePlanned    OpsScheduleStatus = "planned"
	OpsScheduleInspecting OpsScheduleStatus = "inspecting"
	OpsScheduleVerifying  OpsScheduleStatus = "verifying"
	OpsScheduleTreating   OpsScheduleStatus = "treating"
	OpsScheduleDone       OpsScheduleStatus = "done"
	OpsScheduleCancelled  OpsScheduleStatus = "cancelled"
)

type OpsScheduleItem struct {
	ID          string
	RecordID    string
	Status      OpsScheduleStatus
	Priority    OpsPriority
	ScheduledAt string
	UpdatedAt   string
}

// OpsScheduleService plans conservation inspections for records. Each item
// moves through a small state machine: planned -> inspecting -> treating ->
// done, and any state may be cancelled.
type OpsScheduleService struct {
	mu    sync.Mutex
	items map[string]OpsScheduleItem
	order []string
}

func newOpsScheduleService() *OpsScheduleService {
	return &OpsScheduleService{items: map[string]OpsScheduleItem{}, order: []string{}}
}

func (s *OpsScheduleService) Plan(recordID string, priority OpsPriority) (OpsScheduleItem, error) {
	if recordID == "" {
		return OpsScheduleItem{}, fmt.Errorf("%w: schedule record id is required", ErrOpsInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[recordID]; exists {
		return OpsScheduleItem{}, fmt.Errorf("%w: record already planned", ErrOpsConflict)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := OpsScheduleItem{
		ID:          recordID,
		RecordID:    recordID,
		Status:      OpsSchedulePlanned,
		Priority:    priority,
		ScheduledAt: now,
		UpdatedAt:   now,
	}
	s.items[recordID] = item
	s.order = append(s.order, recordID)
	return item, nil
}

func (s *OpsScheduleService) Move(recordID string, target OpsScheduleStatus) (OpsScheduleItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[recordID]
	if !ok {
		return OpsScheduleItem{}, fmt.Errorf("%w: schedule %s not found", ErrOpsNotFound, recordID)
	}
	if !opsScheduleCanMove(item.Status, target) {
		return OpsScheduleItem{}, fmt.Errorf("%w: schedule %s to %s", ErrOpsTransition, item.Status, target)
	}
	item.Status = target
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	s.items[recordID] = item
	return item, nil
}

func scheduleCanMove(from, to OpsScheduleStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case OpsSchedulePlanned:
		return to == OpsScheduleInspecting || to == OpsScheduleCancelled
	case OpsScheduleInspecting:
		return to == OpsScheduleTreating || to == OpsScheduleCancelled
	case OpsScheduleTreating:
		return to == OpsScheduleDone || to == OpsScheduleCancelled
	default:
		return false
	}
}

func (s *OpsScheduleService) List() []OpsScheduleItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]OpsScheduleItem, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.items[id])
	}
	return out
}

func (s *OpsScheduleService) Get(recordID string) (OpsScheduleItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[recordID]
	if !ok {
		return OpsScheduleItem{}, fmt.Errorf("%w: schedule %s not found", ErrOpsNotFound, recordID)
	}
	return item, nil
}

func (s *OpsScheduleService) CountByStatus() map[OpsScheduleStatus]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[OpsScheduleStatus]int{}
	for _, item := range s.items {
		out[item.Status]++
	}
	return out
}

func sortScheduleItems(items []OpsScheduleItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ScheduledAt < items[j].ScheduledAt
	})
}

// StatusSummary groups schedule items into buckets for reporting.
func (s *OpsScheduleService) StatusSummary() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]int{}
	for _, item := range s.items {
		if item.Status == OpsScheduleDone {
			out["done"]++
			continue
		}
		if item.Status == OpsScheduleCancelled {
			out["cancelled"]++
			continue
		}
		if item.Status == OpsScheduleVerifying {
			out["planned"]++
			continue
		}
		if item.Status == OpsScheduleInspecting || item.Status == OpsScheduleTreating {
			out["in_progress"]++
			continue
		}
		out["planned"]++
	}
	return out
}
