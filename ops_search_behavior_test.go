package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testRecord(id, subject, status string) OpsRecord {
	return OpsRecord{
		ID: id, Subject: subject, Owner: "alice", Status: OpsStatus(status), Priority: OpsPriorityNormal,
		Labels: map[string]string{"site": "hall-a"}, UpdatedAt: timeNowOps(),
	}
}

func TestSearchFilterDoesNotCorruptSource(t *testing.T) {
	source := []OpsRecord{
		testRecord("a", "alpha", "queued"),
		testRecord("b", "beta", "active"),
		testRecord("c", "gamma", "queued"),
	}
	queued := opsFilterRecords(source, OpsQuery{Status: OpsStatusQueued})
	if len(queued) != 2 || queued[0].ID != "a" || queued[1].ID != "c" {
		t.Fatalf("unexpected filtered result: %+v", queued)
	}
	if source[0].ID != "a" || source[1].ID != "b" || source[2].ID != "c" {
		t.Fatalf("filtering corrupted the source slice: %+v", source)
	}
}

func TestSearchIndexDoesNotAliasCallerData(t *testing.T) {
	idx := newOpsSearchIndex()
	data := []OpsRecord{testRecord("r1", "original", "queued")}
	idx.Rebuild(data)
	data[0].Subject = "changed-after-rebuild"

	got := idx.Search("", 10)
	if len(got) != 1 || got[0].Subject != "original" {
		t.Fatalf("search index must not alias caller data, got %+v", got)
	}
}

func TestSearchResultsNotRetained(t *testing.T) {
	idx := newOpsSearchIndex()
	idx.Rebuild([]OpsRecord{testRecord("r1", "original", "queued")})

	view := idx.Search("", 10)
	if len(view) != 1 {
		t.Fatalf("unexpected view: %+v", view)
	}
	view[0].Subject = "hacked"

	again := idx.Search("", 10)
	if len(again) != 1 || again[0].Subject != "original" {
		t.Fatalf("mutating a search view must not affect the index: %+v", again)
	}
}

func dashboardItems(app *OpsApp) []OpsRecord {
	items, err := app.Store.List(context.Background())
	if err != nil {
		panic(err)
	}
	return items
}

func TestDashboardSearchResultsRefreshed(t *testing.T) {
	app := newOpsApp()
	seed := []OpsRecord{
		testRecord("d1", "alpha", "queued"),
		testRecord("d2", "beta", "queued"),
	}
	for _, record := range seed {
		if _, err := app.Records.Create(context.Background(), record); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	app.Search.Rebuild(dashboardItems(app))
	app.Cache.Refresh(dashboardItems(app), time.Now())

	ts := httptest.NewServer(NewRouter(NewConservationService(NewArtifactStore()), app))
	defer ts.Close()

	fetch := func() OpsDashboardView {
		resp, err := http.Get(ts.URL + "/api/ops/dashboard")
		if err != nil {
			t.Fatalf("dashboard request: %v", err)
		}
		defer resp.Body.Close()
		var view OpsDashboardView
		if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
			t.Fatalf("decode dashboard: %v", err)
		}
		return view
	}
	first := fetch()
	if len(first.SearchResults) != 2 {
		t.Fatalf("first dashboard should show 2 search results, got %d", len(first.SearchResults))
	}

	third := testRecord("d3", "gamma", "queued")
	if _, err := app.Records.Create(context.Background(), third); err != nil {
		t.Fatalf("create 3rd: %v", err)
	}
	app.Search.Rebuild(dashboardItems(app))
	app.Cache.Refresh(dashboardItems(app), time.Now())

	second := fetch()
	if len(second.SearchResults) != 3 {
		t.Fatalf("dashboard search results must refresh with new data, got %d: %+v", len(second.SearchResults), second.SearchResults)
	}
}
