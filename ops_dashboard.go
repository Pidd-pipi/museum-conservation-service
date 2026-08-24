package main

import (
	"fmt"
	"time"
)

type OpsDashboardView struct {
	Snapshot      OpsSnapshot `json:"snapshot"`
	TopRecords    []OpsRecord `json:"top_records"`
	SearchResults []OpsRecord `json:"search_results"`
	SearchHits    int         `json:"search_hits"`
	Generated     string      `json:"generated"`
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
	// Fetch fresh search results on every build. The search index returns
	// independent copies, so caching the slice between requests would only risk
	// handing one client a slice that a later rebuild mutated.
	results := d.search.Search("", 20)
	view := OpsDashboardView{
		Snapshot:      snapshot,
		TopRecords:    top,
		SearchResults: results,
		SearchHits:    len(results),
		Generated:     now.Format(time.RFC3339Nano),
	}
	return view, nil
}
