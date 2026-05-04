package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func countRequestedSignals(signals map[string]bool) int {
	count := 0
	for _, requested := range signals {
		if requested {
			count++
		}
	}
	return count
}

func joinSignalErrors(signalErrors map[string]string) string {
	if len(signalErrors) == 0 {
		return ""
	}
	keys := make([]string, 0, len(signalErrors))
	for key := range signalErrors {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+": "+strconv.Quote(signalErrors[key]))
	}
	return strings.Join(parts, "; ")
}

func buildPagination(page, limit int, hasNext bool, nextCursor string) Pagination {
	if !hasNext {
		nextCursor = ""
	}
	return Pagination{
		Page:       page,
		Limit:      limit,
		Total:      0,
		TotalPages: 0,
		HasNext:    hasNext,
		NextCursor: nextCursor,
	}
}

func emptyResult(page, limit int) *QueryRunResult {
	result := &QueryRunResult{
		RunID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Status: "complete",
		Pagination: PaginationSet{
			Logs:   buildPagination(page, limit, false, ""),
			Traces: buildPagination(page, limit, false, ""),
		},
		Results: Results{
			Logs:   []any{},
			Traces: []any{},
		},
	}
	result.Summary = QueryRunSummary{}
	return result
}

func requestedSignals(signals []string) map[string]bool {
	if len(signals) == 0 {
		return map[string]bool{"logs": true, "traces": true}
	}
	set := map[string]bool{}
	for _, signal := range signals {
		set[signal] = true
	}
	return set
}

func wrapAny[T any](items []T) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}
