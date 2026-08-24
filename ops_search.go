package main

import (
	"sort"
	"strings"
	"sync"
)

// OpsSearchIndex keeps a flattened, pre-filtered view of records for fast
// keyword search. The index is rebuilt whenever records change.
type OpsSearchIndex struct {
	mu      sync.RWMutex
	items   []OpsRecord
	version int
}

func newOpsSearchIndex() *OpsSearchIndex {
	return &OpsSearchIndex{items: []OpsRecord{}}
}

func (idx *OpsSearchIndex) Rebuild(items []OpsRecord) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.items = items
	sort.SliceStable(idx.items, func(i, j int) bool { return idx.items[i].ID < idx.items[j].ID })
	idx.version++
}

func (idx *OpsSearchIndex) Search(subject string, limit int) []OpsRecord {
	if limit < 1 {
		limit = 20
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(subject))
	out := idx.items
	if needle != "" {
		out = []OpsRecord{}
		for _, item := range idx.items {
			if strings.Contains(strings.ToLower(item.Subject), needle) {
				out = append(out, item)
				if len(out) >= limit {
					break
				}
			}
		}
	}
	return out
}

func (idx *OpsSearchIndex) Version() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.version
}

func (idx *OpsSearchIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.items)
}
