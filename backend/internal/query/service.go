package query

import (
	"context"
	"fmt"
	"log"
	"opendashly/backend/internal/query/builders"
	"opendashly/backend/internal/storage"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Service handles query execution.
type Service struct {
	Storage *storage.Client
	Debug   bool

	cacheInit    sync.Once
	cache        *cacheManager
	runnerInit   sync.Once
	signalRunner signalRunner
}

// Run executes an ad-hoc query and returns results.
func (s *Service) Run(ctx context.Context, req QueryRequest) (*QueryRunResult, error) {
	pagination := normalizeRunPagination(req)

	if s.Debug {
		log.Printf("query.service.run start: page=%d limit=%d offset=%d filters=%d", pagination.page, pagination.limit, pagination.offset, len(req.Filters))
	}
	if s.Storage == nil {
		log.Printf("query.service.run skipped: storage not configured")
		return emptyResult(pagination.page, pagination.limit), nil
	}

	signals := requestedSignals(req.Signals)
	queries, err := buildSignalQueries(req, pagination)
	if err != nil {
		return nil, err
	}
	if s.Debug {
		log.Printf("DEBUG: executing logsQuery: %s", queries.logs)
		log.Printf("query.service.run built queries: logs=%q traces=%q metrics=%q", queries.logs, queries.traces, queries.metrics)
	}
	signalResult := s.getSignalRunner().run(ctx, s.Storage.Conn, queries, signals, pagination.limit)
	signalErrors := signalResult.signalErrors
	requestedCount := countRequestedSignals(signals)
	if len(signalErrors) == requestedCount && requestedCount > 0 {
		return nil, fmt.Errorf("all requested signals failed: %s", joinSignalErrors(signalErrors))
	}

	status := "complete"
	if len(signalErrors) > 0 {
		status = "partial"
		log.Printf("query.service.run partial: failedSignals=%s", joinSignalErrors(signalErrors))
	}

	result := assembleQueryRunResult(pagination, status, signalResult)
	if s.Debug {
		log.Printf("query.service.run complete: runId=%s logs=%d traces=%d metrics=%d", result.RunID, result.Summary.LogCount, result.Summary.TraceCount, result.Summary.MetricCount)
	}
	return result, nil
}

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
			Logs:    buildPagination(page, limit, false, ""),
			Traces:  buildPagination(page, limit, false, ""),
			Metrics: buildPagination(page, limit, false, ""),
		},
		Results: Results{
			Logs:    []any{},
			Traces:  []any{},
			Metrics: []any{},
		},
	}
	result.Summary = QueryRunSummary{}
	return result
}

func requestedSignals(signals []string) map[string]bool {
	if len(signals) == 0 {
		return map[string]bool{"logs": true, "traces": true, "metrics": true}
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

type LogEntry struct {
	Timestamp          time.Time         `json:"timestamp"`
	Severity           string            `json:"severity"`
	Body               string            `json:"body"`
	TraceID            string            `json:"traceId,omitempty"`
	SpanID             string            `json:"spanId,omitempty"`
	ResourceAttributes map[string]string `json:"resourceAttributes,omitempty"`
	LogAttributes      map[string]string `json:"logAttributes,omitempty"`
}

type TraceEntry struct {
	TraceID    string    `json:"traceId"`
	Name       string    `json:"name"`
	Service    string    `json:"service,omitempty"`
	SpanCount  uint64    `json:"spanCount,omitempty"`
	ErrorCount uint64    `json:"errorCount"`
	LastSeen   time.Time `json:"lastSeen,omitempty"`
	DurationMs float64   `json:"durationMs,omitempty"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricSeries struct {
	Name   string        `json:"name"`
	Unit   string        `json:"unit,omitempty"`
	Points []MetricPoint `json:"points"`
}

type metricRow struct {
	Name      string
	Unit      string
	Timestamp time.Time
	Value     float64
}

func fetchLogs(ctx context.Context, conn driver.Conn, query string) ([]LogEntry, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()
	results := []LogEntry{}
	for rows.Next() {
		var row LogEntry
		if err := rows.Scan(&row.Timestamp, &row.Severity, &row.Body, &row.TraceID, &row.SpanID, &row.ResourceAttributes, &row.LogAttributes); err != nil {
			return nil, fmt.Errorf("scan logs: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate logs: %w", err)
	}
	return results, nil
}

func fetchTraces(ctx context.Context, conn driver.Conn, query string) ([]TraceEntry, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query traces: %w", err)
	}
	defer rows.Close()
	results := []TraceEntry{}
	for rows.Next() {
		var row TraceEntry
		if err := rows.Scan(&row.TraceID, &row.Name, &row.Service, &row.SpanCount, &row.ErrorCount, &row.LastSeen, &row.DurationMs); err != nil {
			return nil, fmt.Errorf("scan traces: %w", err)
		}
		if row.DurationMs < 0 {
			row.DurationMs = 0
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate traces: %w", err)
	}
	return results, nil
}

func fetchMetrics(ctx context.Context, conn driver.Conn, query string) ([]MetricSeries, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()
	metricRows := []metricRow{}
	for rows.Next() {
		var row metricRow
		if err := rows.Scan(&row.Name, &row.Unit, &row.Timestamp, &row.Value); err != nil {
			return nil, fmt.Errorf("scan metrics: %w", err)
		}
		metricRows = append(metricRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metrics: %w", err)
	}
	return buildMetricSeries(metricRows), nil
}

func trimToPage[T any](items []T, limit int) ([]T, bool) {
	if limit <= 0 {
		return items, false
	}
	if len(items) > limit {
		return items[:limit], true
	}
	return items, false
}

func (s *Service) getCacheManager() *cacheManager {
	s.cacheInit.Do(func() {
		s.cache = newCacheManager(servicesCacheTTL, attributesCacheTTL)
	})
	return s.cache
}

func (s *Service) getSignalRunner() signalRunner {
	if s.signalRunner != nil {
		return s.signalRunner
	}
	s.runnerInit.Do(func() {
		if s.signalRunner == nil {
			s.signalRunner = defaultSignalOrchestrator()
		}
	})
	return s.signalRunner
}

func (s *Service) getServicesFromCache() ([]string, bool) {
	return s.getCacheManager().getServices()
}

func (s *Service) setServicesCache(values []string) {
	s.getCacheManager().setServices(values)
}

func (s *Service) getAttributesFromCache(search string) ([]string, bool) {
	return s.getCacheManager().getAttributes(search)
}

func (s *Service) setAttributesCache(search string, values []string) {
	s.getCacheManager().setAttributes(search, values)
}

func buildMetricSeries(rows []metricRow) []MetricSeries {
	if len(rows) == 0 {
		return []MetricSeries{}
	}
	grouped := map[string]*MetricSeries{}
	for _, row := range rows {
		key := row.Name + "|" + row.Unit
		series, ok := grouped[key]
		if !ok {
			series = &MetricSeries{Name: row.Name, Unit: row.Unit}
			grouped[key] = series
		}
		series.Points = append(series.Points, MetricPoint{
			Timestamp: row.Timestamp,
			Value:     row.Value,
		})
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	seriesList := make([]MetricSeries, 0, len(keys))
	for _, key := range keys {
		series := grouped[key]
		sort.Slice(series.Points, func(i, j int) bool {
			return series.Points[i].Timestamp.Before(series.Points[j].Timestamp)
		})
		seriesList = append(seriesList, *series)
	}
	return seriesList
}
func (s *Service) GetLogAttributeKeys(ctx context.Context, search string) ([]string, error) {
	if s.Storage == nil {
		return []string{}, nil
	}
	if keys, ok := s.getAttributesFromCache(search); ok {
		return keys, nil
	}

	// Keep this query fast for autocomplete:
	// - read only a bounded recent sample
	// - always include a small seed set
	// This avoids empty suggestions when full-table scans fail or timeout.
	query := "SELECT DISTINCT key FROM (" +
		"SELECT arrayJoin(mapKeys(ResourceAttributes)) AS key FROM (" +
		"SELECT ResourceAttributes FROM telemetry.otel_logs ORDER BY Timestamp DESC LIMIT 50000" +
		") " +
		"UNION ALL " +
		"SELECT arrayJoin(mapKeys(LogAttributes)) AS key FROM (" +
		"SELECT LogAttributes FROM telemetry.otel_logs ORDER BY Timestamp DESC LIMIT 50000" +
		") " +
		"UNION ALL SELECT 'service.name' AS key " +
		"UNION ALL SELECT 'severity' AS key " +
		"UNION ALL SELECT 'trace_id' AS key " +
		"UNION ALL SELECT 'span_id' AS key " +
		"UNION ALL SELECT 'http.method' AS key " +
		"UNION ALL SELECT 'http.route' AS key " +
		"UNION ALL SELECT 'http.status_code' AS key " +
		"UNION ALL SELECT 'error.type' AS key " +
		"UNION ALL SELECT 'error.message' AS key " +
		") WHERE key != ''"
	if search != "" {
		escaped := builders.EscapeLiteral(search)
		query += " AND key ILIKE '%" + escaped + "%'"
	}
	query += " ORDER BY key LIMIT 100"

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query attributes: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(keys) > 0 {
		s.setAttributesCache(search, keys)
		return keys, nil
	}

	seed := []string{
		"service.name",
		"severity",
		"trace_id",
		"span_id",
		"http.method",
		"http.route",
		"http.status_code",
		"error.type",
		"error.message",
	}
	searchLower := strings.ToLower(strings.TrimSpace(search))
	filtered := make([]string, 0, len(seed))
	for _, key := range seed {
		if searchLower == "" || strings.Contains(strings.ToLower(key), searchLower) {
			filtered = append(filtered, key)
		}
	}
	s.setAttributesCache(search, filtered)
	return filtered, nil
}
