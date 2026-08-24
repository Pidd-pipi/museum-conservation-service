package main

import (
	"sort"
	"strings"
)

const opsDomainName = "museum-conservation-service"

type OpsStatus string

const (
	OpsStatusQueued OpsStatus = "queued"
	OpsStatusActive OpsStatus = "active"
	OpsStatusPaused OpsStatus = "paused"
	OpsStatusClosed OpsStatus = "closed"
)

type OpsPriority string

const (
	OpsPriorityLow      OpsPriority = "low"
	OpsPriorityNormal   OpsPriority = "normal"
	OpsPriorityHigh     OpsPriority = "high"
	OpsPriorityCritical OpsPriority = "critical"
)

type OpsRecord struct {
	ID        string
	Subject   string
	Owner     string
	Status    OpsStatus
	Priority  OpsPriority
	Revision  int
	Labels    map[string]string
	CreatedAt string
	UpdatedAt string
}

type OpsRule struct {
	Code           string
	Name           string
	Severity       OpsPriority
	RequiredLabels []string
	Terminal       bool
}

type OpsEvent struct {
	ID       string
	RecordID string
	Type     string
	Actor    string
	At       string
	Details  map[string]string
}

type OpsQuery struct {
	Subject  string
	Status   OpsStatus
	Priority OpsPriority
	Owner    string
	Page     int
	PageSize int
}

type OpsPage struct {
	Items    []OpsRecord
	Page     int
	PageSize int
	Total    int
	HasNext  bool
}

type OpsSnapshot struct {
	Domain      string
	GeneratedAt string
	Records     int
	Active      int
	ByStatus    map[OpsStatus]int
	ByPriority  map[OpsPriority]int
}

// Clone returns a deep copy of the record so callers can mutate the result
// without aliasing shared map state back into the store or cache. The Labels
// map is copied because map values share the same underlying header on a plain
// value copy, which would let edits to the clone leak into the original.
func (r OpsRecord) Clone() OpsRecord {
	out := r
	if r.Labels != nil {
		copied := make(map[string]string, len(r.Labels))
		for key, value := range r.Labels {
			copied[key] = value
		}
		out.Labels = copied
	}
	return out
}

func (r OpsRecord) LabelValue(key string) string { return r.Labels[key] }
func (r OpsRecord) Terminal() bool               { return r.Status == OpsStatusClosed }

func (p OpsPriority) Weight() int {
	switch p {
	case OpsPriorityCritical:
		return 4
	case OpsPriorityHigh:
		return 3
	case OpsPriorityNormal:
		return 2
	default:
		return 1
	}
}

func normalizeOpsRecord(record OpsRecord) OpsRecord {
	record.ID = strings.ToLower(strings.TrimSpace(record.ID))
	record.Subject = strings.Join(strings.Fields(record.Subject), " ")
	record.Owner = strings.TrimSpace(record.Owner)
	if record.Status == "" {
		record.Status = OpsStatusQueued
	}
	if record.Revision < 1 {
		record.Revision = 1
	}
	if record.Labels == nil {
		record.Labels = map[string]string{}
	}
	return record
}

func sortOpsRecords(items []OpsRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority.Weight() != items[j].Priority.Weight() {
			return items[i].Priority.Weight() > items[j].Priority.Weight()
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}

func opsRules() []OpsRule {
	out := make([]OpsRule, 0, 112)
	for _, group := range [][]OpsRule{
		opsRules01(), opsRules02(), opsRules03(), opsRules04(), opsRules05(), opsRules06(), opsRules07(),
		opsRules08(), opsRules09(), opsRules10(), opsRules11(), opsRules12(), opsRules13(), opsRules14(),
	} {
		out = append(out, group...)
	}
	return out
}

// opsDefaultLabels returns the base labels applied to every new record. It
// returns a fresh non-nil map so callers can safely write into it.
func opsDefaultLabels() map[string]string { return map[string]string{} }

// opsMergeLabels returns a new map containing every entry from base overwritten
// by every entry from extra. The inputs are never mutated and nil inputs are
// safe; the result is always non-nil when at least one input is non-nil. This
// avoids the "assignment to entry in nil map" panic that happened when base
// was nil and the caller wrote into it.
func opsMergeLabels(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}
