package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"opendashly/backend/internal/query/builders"
)

type SmartQueryResponse struct {
	SQL      string       `json:"sql"`
	Request  QueryRequest `json:"request"`
	Warnings []string     `json:"warnings,omitempty"`
}

func BuildSmartQuery(prompt string, now time.Time) (*SmartQueryResponse, error) {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return nil, fmt.Errorf("prompt vuoto")
	}
	if len(trimmed) > 500 {
		return nil, fmt.Errorf("prompt troppo lungo")
	}
	if containsUnsafeTokens(trimmed) {
		return nil, fmt.Errorf("prompt contiene token non consentiti")
	}

	minutes := extractRangeMinutes(trimmed)
	from := now.Add(-time.Duration(minutes) * time.Minute)
	to := now

	filters := map[string]string{}
	if serviceName := extractServiceName(trimmed); serviceName != "" {
		filters["service.name"] = serviceName
	}
	if traceID := extractTraceID(trimmed); traceID != "" {
		filters["trace_id"] = traceID
	}

	primary := detectPrimarySignal(trimmed)
	signals := []string{primary}

	request := QueryRequest{
		Signals:   signals,
		TimeRange: TimeRange{From: from, To: to},
		Filters:   filters,
		Page:      1,
		Limit:     100,
	}

	sql := buildPreviewSQL(primary, filters, from, to, request.Limit)

	return &SmartQueryResponse{
		SQL:     sql,
		Request: request,
	}, nil
}

func buildPreviewSQL(signal string, filters map[string]string, from, to time.Time, limit int) string {
	switch signal {
	case "traces":
		return builders.BuildTracesQuery(filters, from, to, limit, 0)
	case "metrics":
		return builders.BuildMetricsQuery(filters, from, to, limit, 0)
	default:
		return builders.BuildLogsQuery(filters, from, to, limit, 0)
	}
}

func containsUnsafeTokens(prompt string) bool {
	lower := strings.ToLower(prompt)
	return strings.Contains(lower, ";") ||
		strings.Contains(lower, "--") ||
		strings.Contains(lower, "/*") ||
		strings.Contains(lower, "*/")
}

func detectPrimarySignal(prompt string) string {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "traccia") || strings.Contains(lower, "tracce") || strings.Contains(lower, "trace") || strings.Contains(lower, "span") {
		return "traces"
	}
	if strings.Contains(lower, "metrica") || strings.Contains(lower, "metriche") || strings.Contains(lower, "metric") {
		return "metrics"
	}
	return "logs"
}

func extractRangeMinutes(prompt string) int {
	lower := strings.ToLower(prompt)
	if minutes := matchDuration(lower, `(\d+)\s*(minuti|minuto|min|minutes|minute|mins)`); minutes > 0 {
		return minutes
	}
	if hours := matchDuration(lower, `(\d+)\s*(ore|ora|hours|hour|hrs)`); hours > 0 {
		return hours * 60
	}
	if strings.Contains(lower, "ultima ora") || strings.Contains(lower, "last hour") {
		return 60
	}
	return 5
}

func matchDuration(prompt, pattern string) int {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(prompt)
	if len(matches) < 2 {
		return 0
	}
	value, err := strconv.Atoi(matches[1])
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func extractServiceName(prompt string) string {
	re := regexp.MustCompile(`(?i)(?:servizio|service)\s*[:=]?\s*([a-zA-Z0-9_.-]+)`)
	matches := re.FindStringSubmatch(prompt)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func extractTraceID(prompt string) string {
	re := regexp.MustCompile(`(?i)(?:traccia|trace)\s*[:=]?\s*([a-f0-9]{8,64})`)
	matches := re.FindStringSubmatch(prompt)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}
