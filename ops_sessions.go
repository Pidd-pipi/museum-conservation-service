package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var opsSessionSequence uint64

type OpsSession struct {
	ID       string `json:"id"`
	RecordID string `json:"record_id"`
	Owner    string `json:"owner"`
	Status   string `json:"status"`
	OpenedAt string `json:"opened_at"`
	ClosedAt string `json:"closed_at"`
}

// OpsSessionManager guards on-site conservation sessions. A session is opened
// for a record before work starts and completed when the work finishes.
type OpsSessionManager struct {
	mu       sync.Mutex
	capacity int
	sessions map[string]OpsSession
	byRecord map[string]string
}

func newOpsSessionManager(capacity int) *OpsSessionManager {
	return &OpsSessionManager{
		capacity: capacity,
		sessions: map[string]OpsSession{},
		byRecord: map[string]string{},
	}
}

func (m *OpsSessionManager) Open(recordID, owner string) (OpsSession, error) {
	if recordID == "" || owner == "" {
		return OpsSession{}, fmt.Errorf("%w: session record and owner are required", ErrOpsInvalid)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) >= m.capacity {
		return OpsSession{}, fmt.Errorf("%w: session capacity reached", ErrOpsPolicy)
	}
	if existing, ok := m.byRecord[recordID]; ok {
		return OpsSession{}, fmt.Errorf("%w: record already has open session %s", ErrOpsConflict, existing)
	}
	session := OpsSession{
		ID:       fmt.Sprintf("sess-%06d", atomic.AddUint64(&opsSessionSequence, 1)),
		RecordID: recordID,
		Owner:    owner,
		Status:   "open",
		OpenedAt: timeNowOps(),
	}
	m.sessions[session.ID] = session
	m.byRecord[recordID] = session.ID
	return session, nil
}

func (m *OpsSessionManager) Complete(id string) (OpsSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return OpsSession{}, fmt.Errorf("%w: session %s not found", ErrOpsNotFound, id)
	}
	if session.Status != "open" {
		return OpsSession{}, fmt.Errorf("%w: session %s is %s", ErrOpsTransition, id, session.Status)
	}
	session.Status = "completed"
	session.ClosedAt = timeNowOps()
	m.sessions[id] = session
	delete(m.byRecord, session.RecordID)
	return session, nil
}

func (m *OpsSessionManager) Abort(id string) (OpsSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return OpsSession{}, fmt.Errorf("%w: session %s not found", ErrOpsNotFound, id)
	}
	if session.Status != "open" {
		return OpsSession{}, fmt.Errorf("%w: session %s is %s", ErrOpsTransition, id, session.Status)
	}
	session.Status = "aborted"
	session.ClosedAt = timeNowOps()
	m.sessions[id] = session
	delete(m.byRecord, session.RecordID)
	return session, nil
}

func (m *OpsSessionManager) List() []OpsSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]OpsSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		out = append(out, session)
	}
	return out
}

func (m *OpsSessionManager) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

// BulkComplete completes several sessions concurrently. It fans out one
// goroutine per session and reports how many sessions were completed.
func (m *OpsSessionManager) BulkComplete(ctx context.Context, ids []string) (int, error) {
	completed := 0
	results := make(chan error, len(ids))
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(sessionID string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				results <- ctx.Err()
				return
			default:
			}
			_, err := m.Complete(sessionID)
			results <- err
		}(id)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err == nil {
			completed++
			continue
		}
		if errors.Is(err, ErrOpsNotFound) {
			continue
		}
		return completed, err
	}
	return completed, nil
}
