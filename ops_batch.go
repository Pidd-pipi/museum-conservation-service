package main

import (
	"context"
	"fmt"
	"sync/atomic"
)

type OpsBatchResult struct {
	BatchID   string
	Requested int
	Applied   int
	Skipped   int
}

// OpsBatchProcessor applies a status transition to a set of records as a
// batch. Each item claims a work unit for the duration of the update; work
// units are released as soon as the item is processed.
type OpsBatchProcessor struct {
	store    *OpsStore
	state    *OpsStateMachine
	manifest *OpsManifestRegistry
	units    chan struct{}
}

func newOpsBatchProcessor(store *OpsStore, state *OpsStateMachine, manifest *OpsManifestRegistry, unitLimit int) *OpsBatchProcessor {
	if unitLimit < 1 {
		unitLimit = 8
	}
	units := make(chan struct{}, unitLimit)
	for i := 0; i < unitLimit; i++ {
		units <- struct{}{}
	}
	return &OpsBatchProcessor{store: store, state: state, manifest: manifest, units: units}
}

func (p *OpsBatchProcessor) Apply(ctx context.Context, owner string, from, to OpsStatus, ids []string) (OpsBatchResult, error) {
	batchID := fmt.Sprintf("batch-%s-%d", owner, atomic.AddUint64(&opsManifestSequence, 1))
	manifest, err := p.manifest.Claim(batchID, owner, len(ids))
	if err != nil {
		return OpsBatchResult{}, err
	}
	defer p.manifest.Release(batchID)

	result := OpsBatchResult{BatchID: batchID, Requested: len(ids)}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		ok, err := p.processOne(ctx, manifest, from, to, id)
		if err != nil {
			return result, err
		}
		if ok {
			result.Applied++
		} else {
			result.Skipped++
		}
	}
	return result, nil
}

// processOne claims a work unit for a single item and releases it before
// returning, so the batch never accumulates open units.
func (p *OpsBatchProcessor) processOne(ctx context.Context, manifest OpsManifest, from, to OpsStatus, id string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-p.units:
	}
	release := func() { p.units <- struct{}{} }
	defer release()

	record, err := p.store.Get(ctx, id)
	if err != nil {
		if err == ErrOpsNotFound {
			return false, nil
		}
		return false, err
	}
	if record.Status != from {
		return false, nil
	}
	if err := p.state.Move(record.Status, to, "batch"); err != nil {
		return false, err
	}
	record.Status = to
	if err := p.store.Update(ctx, record, record.Revision); err != nil {
		return false, err
	}
	_ = p.manifest.Progress(manifest.BatchID, manifest.Done+1)
	return true, nil
}
