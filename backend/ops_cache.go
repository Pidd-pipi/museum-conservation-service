package main

import (
	"sort"
	"sync"
	"time"
)

// OpsSnapshotCache caches the computed operations snapshot for a short TTL so
// repeated snapshot requests do not rescan the whole store every time.
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
	c.generated = at
	c.total = len(items)
	c.active = 0
	c.byStatus = map[OpsStatus]int{}
	c.byPriority = map[OpsPriority]int{}
	for _, item := range items {
		c.byStatus[item.Status]++
		c.byPriority[item.Priority]++
		if item.Status == OpsStatusActive {
			c.active++
		}
	}
	c.top = c.top[:0]
	for _, item := range items {
		c.top = append(c.top, item.Clone())
	}
	sort.SliceStable(c.top, func(i, j int) bool {
		if c.top[i].Priority.Weight() != c.top[j].Priority.Weight() {
			return c.top[i].Priority.Weight() > c.top[j].Priority.Weight()
		}
		return c.top[i].UpdatedAt > c.top[j].UpdatedAt
	})
}

// Get returns a deep copy of the cached snapshot. The second return value is
// false when the cache is older than the TTL or was never refreshed.
func (c *OpsSnapshotCache) Get(now time.Time) (OpsSnapshot, []OpsRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.generated.IsZero() || now.Sub(c.generated) > c.ttl {
		return OpsSnapshot{}, nil, false
	}
	snapshot := OpsSnapshot{
		Domain:      opsDomainName,
		GeneratedAt: c.generated.Format(time.RFC3339Nano),
		Records:     c.total,
		Active:      c.active,
		ByStatus:    map[OpsStatus]int{},
		ByPriority:  map[OpsPriority]int{},
	}
	for status, count := range c.byStatus {
		snapshot.ByStatus[status] = count
	}
	for priority, count := range c.byPriority {
		snapshot.ByPriority[priority] = count
	}
	top := make([]OpsRecord, 0, len(c.top))
	for _, item := range c.top {
		top = append(top, item.Clone())
	}
	return snapshot, top, true
}
