package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type OpsPolicy struct {
	RequireOwner  bool
	RequiredLabel string
	MaxActive     int
}
type OpsService struct {
	store  *OpsStore
	audit  *OpsAudit
	state  *OpsStateMachine
	policy OpsPolicy
	clock  OpsClock
	notes  *OpsNoteStore
}

func newOpsService(seed []OpsRecord) *OpsService {
	return newOpsServiceWith(newOpsStore(seed), newOpsAudit(), newOpsStateMachine(), newOpsClock(), newOpsNoteStore(nil))
}

func newOpsServiceWith(store *OpsStore, audit *OpsAudit, state *OpsStateMachine, clock OpsClock, notes *OpsNoteStore) *OpsService {
	return &OpsService{store: store, audit: audit, state: state, policy: OpsPolicy{RequireOwner: true, RequiredLabel: "site", MaxActive: 1000}, clock: clock, notes: notes}
}
func (p OpsPolicy) Check(record OpsRecord) error {
	if p.RequireOwner && strings.TrimSpace(record.Owner) == "" {
		return fmt.Errorf("%w: owner required", ErrOpsPolicy)
	}
	if p.RequiredLabel != "" && record.LabelValue(p.RequiredLabel) == "" {
		return fmt.Errorf("%w: %s label required", ErrOpsPolicy, p.RequiredLabel)
	}
	if record.Priority == "" {
		return fmt.Errorf("%w: priority required", ErrOpsPolicy)
	}
	return nil
}
func (s *OpsService) Create(ctx context.Context, record OpsRecord) (OpsRecord, error) {
	record = normalizeOpsRecord(record)
	if err := s.policy.Check(record); err != nil {
		return OpsRecord{}, err
	}
	record.CreatedAt = s.clock.Stamp()
	record.UpdatedAt = record.CreatedAt
	if err := s.store.Put(ctx, record); err != nil {
		return OpsRecord{}, wrapOps("create", "store.put", err)
	}
	s.audit.Add(record.ID, "created", record.Owner)
	return record, nil
}
func (s *OpsService) Get(ctx context.Context, id string) (OpsRecord, error) {
	return s.store.Get(ctx, id)
}
func (s *OpsService) Search(ctx context.Context, q OpsQuery) (OpsPage, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return OpsPage{}, err
	}
	filtered := make([]OpsRecord, 0, len(items))
	for _, item := range items {
		if opsMatch(item, q) {
			filtered = append(filtered, item)
		}
	}
	sortOpsRecords(filtered)
	q = opsQueryDefaults(q)
	start, end := opsBounds(len(filtered), q.Page, q.PageSize)
	// Copy the page into a slice with exact length. filtered[start:end] would
	// alias the full filtered backing array, leaking all the records on other
	// pages through the returned slice's capacity; a fresh copy keeps each
	// search result self-contained.
	pageItems := make([]OpsRecord, end-start)
	copy(pageItems, filtered[start:end])
	return OpsPage{Items: pageItems, Page: q.Page, PageSize: q.PageSize, Total: len(filtered), HasNext: end < len(filtered)}, nil
}
func (s *OpsService) Transition(ctx context.Context, id string, expected int, target OpsStatus, actor string) (OpsRecord, error) {
	ctx, cancel := opsContext(ctx, 3*time.Second)
	defer cancel()
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return OpsRecord{}, err
	}
	if expected > 0 && expected != record.Revision {
		return OpsRecord{}, ErrOpsConflict
	}
	if err := s.state.Move(record.Status, target, "operator update"); err != nil {
		return OpsRecord{}, err
	}
	record.Status = target
	if err := s.store.Update(ctx, record, expected); err != nil {
		return OpsRecord{}, err
	}
	s.audit.Add(record.ID, "status_changed", actor)
	return record, nil
}
func (s *OpsService) Audit(id string) []OpsEvent { return s.audit.For(id) }
func (s *OpsService) Snapshot() OpsSnapshot {
	items, _ := s.store.List(context.Background())
	out := OpsSnapshot{Domain: opsDomainName, GeneratedAt: s.clock.Stamp(), ByStatus: map[OpsStatus]int{}, ByPriority: map[OpsPriority]int{}}
	for _, i := range items {
		out.Records++
		out.ByStatus[i.Status]++
		out.ByPriority[i.Priority]++
		if i.Status == OpsStatusActive {
			out.Active++
		}
	}
	return out
}
func (s *OpsService) AddNote(recordID, author, text string) (OpsNote, error) {
	note, err := s.notes.Add(recordID, author, text)
	if err != nil {
		return OpsNote{}, err
	}
	s.audit.Add(recordID, "note_added", author)
	return note, nil
}

func (s *OpsService) Domain() string { return opsDomainName }
func (s *OpsService) Count() int     { return s.store.Count() }
func timeNowOps() string             { return time.Now().UTC().Format(time.RFC3339Nano) }
