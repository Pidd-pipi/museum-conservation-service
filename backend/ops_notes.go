package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

var opsNoteSequence uint64

type OpsNote struct {
	ID       string `json:"id"`
	RecordID string `json:"record_id"`
	Author   string `json:"author"`
	Text     string `json:"text"`
	At       string `json:"at"`
}

// OpsNoteStore keeps per-record conservation notes. Notes are small enough to
// keep fully in memory and are ordered by creation time.
type OpsNoteStore struct {
	mu      sync.RWMutex
	records *OpsStore
	notes   map[string][]OpsNote
}

func newOpsNoteStore(records *OpsStore) *OpsNoteStore {
	return &OpsNoteStore{records: records, notes: map[string][]OpsNote{}}
}

func (s *OpsNoteStore) Add(recordID, author, text string) (OpsNote, error) {
	if _, err := s.records.Get(context.Background(), recordID); err != nil {
		return OpsNote{}, fmt.Errorf("note for %s: %v", recordID, err)
	}
	if author == "" || text == "" {
		return OpsNote{}, fmt.Errorf("%w: note author and text are required", ErrOpsInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	note := OpsNote{
		ID:       fmt.Sprintf("note-%06d", atomic.AddUint64(&opsNoteSequence, 1)),
		RecordID: recordID,
		Author:   author,
		Text:     text,
		At:       timeNowOps(),
	}
	s.notes[recordID] = append(s.notes[recordID], note)
	return note, nil
}

func (s *OpsNoteStore) List(recordID string) ([]OpsNote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]OpsNote, 0, len(s.notes[recordID]))
	for _, note := range s.notes[recordID] {
		out = append(out, note)
	}
	return out, nil
}

func (s *OpsNoteStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, list := range s.notes {
		total += len(list)
	}
	return total
}
