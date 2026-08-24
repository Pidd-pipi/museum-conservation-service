package main

import (
	"context"
	"fmt"
	"strings"
)

// OpsExportService renders records into a plain-text work sheet. Export can be
// large, so it honours context cancellation between records.
type OpsExportService struct {
	store *OpsStore
}

func newOpsExportService(store *OpsStore) *OpsExportService {
	return &OpsExportService{store: store}
}

func (s *OpsExportService) Export(ctx context.Context, status OpsStatus, limit int) (string, error) {
	if limit < 1 {
		limit = 500
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	exported := 0
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if status != "" && item.Status != status {
			continue
		}
		builder.WriteString(fmt.Sprintf("%s|%s|%s|%s|%d\n", item.ID, item.Subject, item.Owner, item.Status, item.Revision))
		exported++
		if exported >= limit {
			break
		}
	}
	return builder.String(), nil
}

func (s *OpsExportService) ExportCount(ctx context.Context) (int, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}
