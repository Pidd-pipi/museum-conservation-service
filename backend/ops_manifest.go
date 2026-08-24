package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var opsManifestSequence uint64

type OpsManifest struct {
	ID        string
	BatchID   string
	Owner     string
	Scope     int
	Done      int
	CreatedAt string
}

// OpsManifestRegistry tracks batch work manifests. A manifest must be claimed
// before a batch touches records and released when the batch finishes.
type OpsManifestRegistry struct {
	mu        sync.Mutex
	manifests map[string]OpsManifest
}

func newOpsManifestRegistry() *OpsManifestRegistry {
	return &OpsManifestRegistry{manifests: map[string]OpsManifest{}}
}

func (r *OpsManifestRegistry) Claim(batchID, owner string, scope int) (OpsManifest, error) {
	if batchID == "" || owner == "" {
		return OpsManifest{}, fmt.Errorf("%w: batch owner and id are required", ErrOpsInvalid)
	}
	if scope < 1 {
		return OpsManifest{}, fmt.Errorf("%w: manifest scope must be positive", ErrOpsInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.manifests[batchID]; exists {
		return OpsManifest{}, fmt.Errorf("%w: batch %s already claimed", ErrOpsConflict, batchID)
	}
	manifest := OpsManifest{
		ID:        fmt.Sprintf("mf-%06d", atomic.AddUint64(&opsManifestSequence, 1)),
		BatchID:   batchID,
		Owner:     owner,
		Scope:     scope,
		Done:      0,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	r.manifests[batchID] = manifest
	return manifest, nil
}

func (r *OpsManifestRegistry) Progress(batchID string, done int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	manifest, ok := r.manifests[batchID]
	if !ok {
		return fmt.Errorf("%w: manifest %s not found", ErrOpsNotFound, batchID)
	}
	if done > manifest.Scope {
		return fmt.Errorf("%w: manifest progress exceeds scope", ErrOpsInvalid)
	}
	manifest.Done = done
	r.manifests[batchID] = manifest
	return nil
}

func (r *OpsManifestRegistry) Release(batchID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.manifests[batchID]; !ok {
		return fmt.Errorf("%w: manifest %s not found", ErrOpsNotFound, batchID)
	}
	return nil
}

func (r *OpsManifestRegistry) Get(batchID string) (OpsManifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	manifest, ok := r.manifests[batchID]
	if !ok {
		return OpsManifest{}, fmt.Errorf("%w: manifest %s not found", ErrOpsNotFound, batchID)
	}
	return manifest, nil
}

func (r *OpsManifestRegistry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.manifests)
}
