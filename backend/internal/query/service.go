package query

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"opendashly/backend/internal/query/builders"
	"opendashly/backend/internal/storage"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Service handles query execution.
type Service struct {
	Storage *storage.Client
	Debug   bool
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

	if s.Debug {
		log.Printf("query.service.run start: page=%d limit=%d offset=%d filters=%d", page, limit, offset, len(req.Filters))
	}
	if s.Storage == nil {
		log.Printf("query.service.run skipped: storage not configured")
		return emptyResult(page, limit), nil
	}

	signals := requestedSignals(req.Signals)
	logsQuery := builders.BuildLogsQuery(req.Filters, req.FilterList, req.TimeRange.From, req.TimeRange.To, limit, offset)
	tracesQuery := builders.BuildTracesQuery(req.Filters, req.FilterList, req.TimeRange.From, req.TimeRange.To, limit, offset)
	metricsQuery := builders.BuildMetricsQuery(req.Filters, req.FilterList, req.TimeRange.From, req.TimeRange.To, limit, offset)
	if s.Debug {
		log.Printf("DEBUG: executing logsQuery: %s", logsQuery)
		log.Printf("query.service.run built queries: logs=%q traces=%q metrics=%q", logsQuery, tracesQuery, metricsQuery)
	}

	var logs []LogEntry
	var traces []TraceEntry
	var metrics []MetricSeries
	var err error
	if signals["logs"] {
		logs, err = fetchLogs(ctx, s.Storage.Conn, logsQuery)
		if err != nil {
			return nil, err
		}
	}
	if signals["traces"] {
		traces, err = fetchTraces(ctx, s.Storage.Conn, tracesQuery)
		if err != nil {
			return nil, err
		}
	}
	if signals["metrics"] {
		metrics, err = fetchMetrics(ctx, s.Storage.Conn, metricsQuery)
		if err != nil {
			return nil, err
		}
	}

	result := &QueryRunResult{
		RunID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Status: "complete",
		Pagination: PaginationSet{
			Logs:    buildPagination(page, limit, len(logs)),
			Traces:  buildPagination(page, limit, len(traces)),
			Metrics: buildPagination(page, limit, len(metrics)),
		},
		Results: Results{
			Logs:    wrapAny(logs),
			Traces:  wrapAny(traces),
			Metrics: wrapAny(metrics),
		},
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

func buildPagination(page, limit, total int) Pagination {
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}
	return Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

func emptyResult(page, limit int) *QueryRunResult {
	result := &QueryRunResult{
		RunID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Status: "complete",
		Pagination: PaginationSet{
			Logs:    buildPagination(page, limit, 0),
			Traces:  buildPagination(page, limit, 0),
			Metrics: buildPagination(page, limit, 0),
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

	// Limit to last 24h to avoid scanning too much
	query := "SELECT DISTINCT arrayJoin(mapKeys(LogAttributes)) as key FROM telemetry.otel_logs WHERE Timestamp > now() - INTERVAL 24 HOUR"
	if search != "" {
		escaped := builders.EscapeLiteral(search)
		query += " AND key ILIKE '%" + escaped + "%'"
	}
	query += " ORDER BY key LIMIT 50"

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
	return keys, nil
}
