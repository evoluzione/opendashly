package query

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"opendashly/backend/internal/query/builders"
	"opendashly/backend/internal/storage"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Service handles query execution.
type Service struct {
	Storage *storage.Client
	Debug   bool

	cacheInit sync.Once
	cache     *cacheManager
}

// Run executes an ad-hoc query and returns results.
func (s *Service) Run(ctx context.Context, req QueryRequest) (*QueryRunResult, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 100
	}
	offset := (page - 1) * limit
	readLimit := limit + 1

	if s.Debug {
		log.Printf("query.service.run start: page=%d limit=%d offset=%d filters=%d", page, limit, offset, len(req.Filters))
	}
	if s.Storage == nil {
		log.Printf("query.service.run skipped: storage not configured")
		return emptyResult(page, limit), nil
	}

	logsCursor, err := decodeLogsCursor(req.LogsCursor)
	if err != nil {
		return nil, fmt.Errorf("decode logs cursor: %w", err)
	}
	tracesCursor, err := decodeTracesCursor(req.TracesCursor)
	if err != nil {
		return nil, fmt.Errorf("decode traces cursor: %w", err)
	}

	signals := requestedSignals(req.Signals)
	logsQuery := builders.BuildLogsQuery(req.Filters, req.FilterList, req.TimeRange.From, req.TimeRange.To, readLimit, offset, logsCursor)
	tracesQuery := builders.BuildTracesQuery(req.Filters, req.FilterList, req.TimeRange.From, req.TimeRange.To, readLimit, offset, tracesCursor)
	metricsQuery := builders.BuildMetricsQuery(req.Filters, req.FilterList, req.TimeRange.From, req.TimeRange.To, readLimit, offset)
	if s.Debug {
		log.Printf("DEBUG: executing logsQuery: %s", logsQuery)
		log.Printf("query.service.run built queries: logs=%q traces=%q metrics=%q", logsQuery, tracesQuery, metricsQuery)
	}

	var logs []LogEntry
	var traces []TraceEntry
	var metrics []MetricSeries

	var logsHasNext, tracesHasNext, metricsHasNext bool
	var logsNextCursor, tracesNextCursor string

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	signalErrors := map[string]string{}

	setSignalError := func(signal string, err error) {
		if err == nil {
			return
		}
		mu.Lock()
		signalErrors[signal] = err.Error()
		mu.Unlock()
	}

	if signals["logs"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			signalLogs, err := fetchLogs(ctx, s.Storage.Conn, logsQuery)
			if err != nil {
				setSignalError("logs", err)
				return
			}
			signalLogs, signalLogsHasNext := trimToPage(signalLogs, limit)
			signalLogsNextCursor := ""
			if signalLogsHasNext {
				signalLogsNextCursor, err = encodeLogsCursor(signalLogs[len(signalLogs)-1])
				if err != nil {
					setSignalError("logs", fmt.Errorf("encode logs cursor: %w", err))
					return
				}
			}

			mu.Lock()
			logs = signalLogs
			logsHasNext = signalLogsHasNext
			logsNextCursor = signalLogsNextCursor
			mu.Unlock()
		}()
	}
	if signals["traces"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			signalTraces, err := fetchTraces(ctx, s.Storage.Conn, tracesQuery)
			if err != nil {
				setSignalError("traces", err)
				return
			}
			signalTraces, signalTracesHasNext := trimToPage(signalTraces, limit)
			signalTracesNextCursor := ""
			if signalTracesHasNext {
				signalTracesNextCursor, err = encodeTracesCursor(signalTraces[len(signalTraces)-1])
				if err != nil {
					setSignalError("traces", fmt.Errorf("encode traces cursor: %w", err))
					return
				}
			}

			mu.Lock()
			traces = signalTraces
			tracesHasNext = signalTracesHasNext
			tracesNextCursor = signalTracesNextCursor
			mu.Unlock()
		}()
	}
	if signals["metrics"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			signalMetrics, err := fetchMetrics(ctx, s.Storage.Conn, metricsQuery)
			if err != nil {
				setSignalError("metrics", err)
				return
			}
			signalMetrics, signalMetricsHasNext := trimToPage(signalMetrics, limit)

			mu.Lock()
			metrics = signalMetrics
			metricsHasNext = signalMetricsHasNext
			mu.Unlock()
		}()
	}
	wg.Wait()
	requestedCount := countRequestedSignals(signals)
	if len(signalErrors) == requestedCount && requestedCount > 0 {
		return nil, fmt.Errorf("all requested signals failed: %s", joinSignalErrors(signalErrors))
	}

	status := "complete"
	if len(signalErrors) > 0 {
		status = "partial"
		log.Printf("query.service.run partial: failedSignals=%s", joinSignalErrors(signalErrors))
	}

	result := &QueryRunResult{
		RunID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Status: status,
		Pagination: PaginationSet{
			Logs:    buildPagination(page, limit, logsHasNext, logsNextCursor),
			Traces:  buildPagination(page, limit, tracesHasNext, tracesNextCursor),
			Metrics: buildPagination(page, limit, metricsHasNext, ""),
		},
		Results: Results{
			Logs:    wrapAny(logs),
			Traces:  wrapAny(traces),
			Metrics: wrapAny(metrics),
		},
	}
	if len(signalErrors) > 0 {
		result.SignalErrors = signalErrors
	}
	result.Summary = QueryRunSummary{
		LogCount:    len(logs),
		TraceCount:  len(traces),
		MetricCount: len(metrics),
	}
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

type logsCursorPayload struct {
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"traceId"`
	SpanID    string    `json:"spanId"`
}

type tracesCursorPayload struct {
	LastSeen time.Time `json:"lastSeen"`
	TraceID  string    `json:"traceId"`
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

func decodeLogsCursor(value string) (*builders.LogsPageCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var payload logsCursorPayload
	if err := decodeCursor(value, &payload); err != nil {
		return nil, err
	}
	if payload.Timestamp.IsZero() {
		return nil, nil
	}
	return &builders.LogsPageCursor{
		Timestamp: payload.Timestamp,
		TraceID:   payload.TraceID,
		SpanID:    payload.SpanID,
	}, nil
}

func decodeTracesCursor(value string) (*builders.TracesPageCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var payload tracesCursorPayload
	if err := decodeCursor(value, &payload); err != nil {
		return nil, err
	}
	if payload.LastSeen.IsZero() {
		return nil, nil
	}
	return &builders.TracesPageCursor{
		LastSeen: payload.LastSeen,
		TraceID:  payload.TraceID,
	}, nil
}

func encodeLogsCursor(last LogEntry) (string, error) {
	return encodeCursor(logsCursorPayload{
		Timestamp: last.Timestamp,
		TraceID:   last.TraceID,
		SpanID:    last.SpanID,
	})
}

func encodeTracesCursor(last TraceEntry) (string, error) {
	return encodeCursor(tracesCursorPayload{
		LastSeen: last.LastSeen,
		TraceID:  last.TraceID,
	})
}

func encodeCursor(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(encoded string, dest any) error {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func (s *Service) getCacheManager() *cacheManager {
	s.cacheInit.Do(func() {
		s.cache = newCacheManager(servicesCacheTTL, attributesCacheTTL)
	})
	return s.cache
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
