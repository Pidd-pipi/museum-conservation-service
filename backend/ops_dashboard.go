package main

import (
	"fmt"
	"time"
)

type OpsDashboardView struct {
	Snapshot   OpsSnapshot `json:"snapshot"`
	TopRecords []OpsRecord `json:"top_records"`
	SearchHits int         `json:"search_hits"`
	Generated  string      `json:"generated"`
}

// OpsDashboard assembles the operations overview from the snapshot cache and
// the search index.
type OpsDashboard struct {
	cache  *OpsSnapshotCache
	search *OpsSearchIndex
}

func newOpsDashboard(cache *OpsSnapshotCache, search *OpsSearchIndex) *OpsDashboard {
	return &OpsDashboard{cache: cache, search: search}
}

func (d *OpsDashboard) Build(now time.Time) (OpsDashboardView, error) {
	snapshot, top, fresh := d.cache.Get(now)
	if !fresh {
		return OpsDashboardView{}, fmt.Errorf("%w: snapshot cache is stale", ErrOpsNotFound)
	}
	view := OpsDashboardView{
		Snapshot:   snapshot,
		TopRecords: top,
		SearchHits: len(d.search.Search("", 20)),
		Generated:  now.Format(time.RFC3339Nano),
	}
	return view, nil
}
