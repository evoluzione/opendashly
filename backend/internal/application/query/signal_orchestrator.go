package query

import (
	"context"
	"fmt"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type logsFetcher func(context.Context, driver.Conn, string) ([]LogEntry, error)
type tracesFetcher func(context.Context, driver.Conn, string) ([]TraceEntry, error)
type metricsFetcher func(context.Context, driver.Conn, string) ([]MetricSeries, error)

type signalQueries struct {
	logs    string
	traces  string
	metrics string
}

type signalExecutionResult struct {
	logs             []LogEntry
	traces           []TraceEntry
	metrics          []MetricSeries
	logsHasNext      bool
	tracesHasNext    bool
	metricsHasNext   bool
	logsNextCursor   string
	tracesNextCursor string
	signalErrors     map[string]string
}

type signalRunner interface {
	run(context.Context, driver.Conn, signalQueries, map[string]bool, int) signalExecutionResult
}

type signalOrchestrator struct {
	fetchLogs    logsFetcher
	fetchTraces  tracesFetcher
	fetchMetrics metricsFetcher
}

func newSignalOrchestrator(fetchLogs logsFetcher, fetchTraces tracesFetcher, fetchMetrics metricsFetcher) *signalOrchestrator {
	return &signalOrchestrator{
		fetchLogs:    fetchLogs,
		fetchTraces:  fetchTraces,
		fetchMetrics: fetchMetrics,
	}
}

func defaultSignalOrchestrator() *signalOrchestrator {
	return newSignalOrchestrator(fetchLogs, fetchTraces, fetchMetrics)
}

func (o *signalOrchestrator) run(ctx context.Context, conn driver.Conn, queries signalQueries, signals map[string]bool, limit int) signalExecutionResult {
	result := signalExecutionResult{signalErrors: map[string]string{}}
	if o == nil {
		return result
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	if signals["logs"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.executeLogs(ctx, conn, queries.logs, limit, &mu, &result)
		}()
	}

	if signals["traces"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.executeTraces(ctx, conn, queries.traces, limit, &mu, &result)
		}()
	}

	if signals["metrics"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.executeMetrics(ctx, conn, queries.metrics, limit, &mu, &result)
		}()
	}

	wg.Wait()
	return result
}

func (o *signalOrchestrator) executeLogs(ctx context.Context, conn driver.Conn, query string, limit int, mu *sync.Mutex, result *signalExecutionResult) {
	signalLogs, err := o.fetchLogs(ctx, conn, query)
	if err != nil {
		setSignalError(mu, result, "logs", err)
		return
	}
	signalLogs, signalLogsHasNext := trimToPage(signalLogs, limit)
	signalLogsNextCursor := ""
	if signalLogsHasNext {
		signalLogsNextCursor, err = encodeLogsCursor(signalLogs[len(signalLogs)-1])
		if err != nil {
			setSignalError(mu, result, "logs", fmt.Errorf("encode logs cursor: %w", err))
			return
		}
	}

	mu.Lock()
	result.logs = signalLogs
	result.logsHasNext = signalLogsHasNext
	result.logsNextCursor = signalLogsNextCursor
	mu.Unlock()
}

func (o *signalOrchestrator) executeTraces(ctx context.Context, conn driver.Conn, query string, limit int, mu *sync.Mutex, result *signalExecutionResult) {
	signalTraces, err := o.fetchTraces(ctx, conn, query)
	if err != nil {
		setSignalError(mu, result, "traces", err)
		return
	}
	signalTraces, signalTracesHasNext := trimToPage(signalTraces, limit)
	signalTracesNextCursor := ""
	if signalTracesHasNext {
		signalTracesNextCursor, err = encodeTracesCursor(signalTraces[len(signalTraces)-1])
		if err != nil {
			setSignalError(mu, result, "traces", fmt.Errorf("encode traces cursor: %w", err))
			return
		}
	}

	mu.Lock()
	result.traces = signalTraces
	result.tracesHasNext = signalTracesHasNext
	result.tracesNextCursor = signalTracesNextCursor
	mu.Unlock()
}

func (o *signalOrchestrator) executeMetrics(ctx context.Context, conn driver.Conn, query string, limit int, mu *sync.Mutex, result *signalExecutionResult) {
	signalMetrics, err := o.fetchMetrics(ctx, conn, query)
	if err != nil {
		setSignalError(mu, result, "metrics", err)
		return
	}
	signalMetrics, signalMetricsHasNext := trimToPage(signalMetrics, limit)

	mu.Lock()
	result.metrics = signalMetrics
	result.metricsHasNext = signalMetricsHasNext
	mu.Unlock()
}

func setSignalError(mu *sync.Mutex, result *signalExecutionResult, signal string, err error) {
	if err == nil {
		return
	}
	mu.Lock()
	result.signalErrors[signal] = err.Error()
	mu.Unlock()
}
