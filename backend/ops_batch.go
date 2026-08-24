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
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-p.units:
		}
		defer func() { p.units <- struct{}{} }()

		record, err := p.store.Get(ctx, id)
		if err != nil {
			if err == ErrOpsNotFound {
				result.Skipped++
				continue
			}
			return result, err
		}
		if record.Status != from {
			result.Skipped++
			continue
		}
		if err := p.state.Move(record.Status, to, "batch"); err != nil {
			result.Skipped++
			continue
		}
		record.Status = to
		if err := p.store.Update(ctx, record, record.Revision); err != nil {
			return result, err
		}
		_ = p.manifest.Progress(manifest.BatchID, manifest.Done+1)
		result.Applied++
	}
	return result, nil
}
