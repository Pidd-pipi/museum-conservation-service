package main

import (
	"sort"
	"sync"
	"time"
)

// OpsSnapshotCache caches the computed operations snapshot for a short TTL so
// repeated snapshot requests do not rescan the whole store every time.
//
// The cache is refreshed by a background worker and read by HTTP handlers, so
// every access is guarded by mu and Get returns defensive copies. Callers
// therefore never observe a snapshot that is being rebuilt underneath them,
// which previously caused the top list to shift between refreshes (and the
// occasional 500 from a concurrent map/slice read).
type OpsSnapshotCache struct {
	mu         sync.RWMutex
	ttl        time.Duration
	generated  time.Time
	total      int
	active     int
	byStatus   map[OpsStatus]int
	byPriority map[OpsPriority]int
	top        []OpsRecord
}

func newOpsSnapshotCache(ttl time.Duration) *OpsSnapshotCache {
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &OpsSnapshotCache{
		ttl:        ttl,
		byStatus:   map[OpsStatus]int{},
		byPriority: map[OpsPriority]int{},
		top:        []OpsRecord{},
	}
}

func (c *OpsSnapshotCache) Refresh(items []OpsRecord, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	byStatus := make(map[OpsStatus]int, len(c.byStatus))
	byPriority := make(map[OpsPriority]int, len(c.byPriority))
	top := make([]OpsRecord, 0, len(items))
	active := 0
	for _, item := range items {
		byStatus[item.Status]++
		byPriority[item.Priority]++
		if item.Status == OpsStatusActive {
			active++
		}
		top = append(top, item.Clone())
	}
	sort.SliceStable(top, func(i, j int) bool {
		if top[i].Priority.Weight() != top[j].Priority.Weight() {
			return top[i].Priority.Weight() > top[j].Priority.Weight()
		}
		return top[i].UpdatedAt > top[j].UpdatedAt
	})
	c.generated = at
	c.total = len(items)
	c.active = active
	c.byStatus = byStatus
	c.byPriority = byPriority
	c.top = top
}

// Get returns a deep copy of the cached snapshot. The third return value is
// false when the cache is older than the TTL or was never refreshed, so the
// caller should recompute the snapshot itself in that case.
func (c *OpsSnapshotCache) Get(now time.Time) (OpsSnapshot, []OpsRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.generated.IsZero() || now.Sub(c.generated) > c.ttl {
		return OpsSnapshot{}, nil, false
	}
	byStatus := make(map[OpsStatus]int, len(c.byStatus))
	for status, count := range c.byStatus {
		byStatus[status] = count
	}
	byPriority := make(map[OpsPriority]int, len(c.byPriority))
	for priority, count := range c.byPriority {
		byPriority[priority] = count
	}
	top := make([]OpsRecord, len(c.top))
	for i := range c.top {
		top[i] = c.top[i].Clone()
	}
	snapshot := OpsSnapshot{
		Domain:      opsDomainName,
		GeneratedAt: c.generated.Format(time.RFC3339Nano),
		Records:     c.total,
		Active:      c.active,
		ByStatus:    byStatus,
		ByPriority:  byPriority,
	}
	return snapshot, top, true
}
