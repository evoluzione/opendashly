package query

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"opendashly/backend/internal/ai"
	"opendashly/backend/internal/query/builders"

	openai "github.com/sashabaranov/go-openai"
)

type SmartQueryResponse struct {
	SQL      string       `json:"sql"`
	Request  QueryRequest `json:"request"`
	Warnings []string     `json:"warnings,omitempty"`
}

func BuildSmartQuery(prompt, contextType string, settings *ai.Settings, now time.Time) (*SmartQueryResponse, error) {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return nil, fmt.Errorf("prompt vuoto")
	}
	if len(trimmed) > 500 {
		return nil, fmt.Errorf("prompt troppo lungo")
	}

	// Use AI if API key is present
	if settings.Enabled && settings.APIKey != "" {
		return generateSQLWithAI(trimmed, contextType, settings, now)
	}

	// Fallback to regex-based (legacy)
	return buildLegacySmartQuery(trimmed, now)
}

func generateSQLWithAI(prompt, contextType string, settings *ai.Settings, now time.Time) (*SmartQueryResponse, error) {
	client := openai.NewClient(settings.APIKey)

	var schemaDesc string
	switch contextType {
	case "metrics":
		schemaDesc = MetricsSchemaDescription
	case "logs":
		schemaDesc = LogsSchemaDescription
	case "traces":
		schemaDesc = TracesSchemaDescription
	default: // "auto" or empty
		schemaDesc = fmt.Sprintf("LOGS SCHEMA:\n%s\n\nMETRICS SCHEMA:\n%s\n\nTRACES SCHEMA:\n%s", LogsSchemaDescription, MetricsSchemaDescription, TracesSchemaDescription)
	}

	systemPrompt := fmt.Sprintf(`You are a ClickHouse SQL expert for OpenTelemetry data.
Your goal is to generate a VALID ClickHouse SQL query based on the user request.
Return ONLY the SQL string. No markdown, no explanations.

Current Time: %s

Schema Context:
%s

Rules:
1. Use the provided table names.
2. For specific time ranges, use appropriate WHERE clauses with now() or specific timestamps.
3. If no time range is specified, default to the last 15 minutes.
4. Text comparisons should be case-insensitive if appropriate (ilike).
5. Determine if the user is asking for logs, metrics, or traces and use the appropriate table.
6. Return ONLY SQL.
`, now.Format(time.RFC3339), schemaDesc)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.2, // Low temperature for deterministic code generation
		},
	)

	if err != nil {
		return nil, fmt.Errorf("errore AI: %v", err)
	}

	sql := strings.TrimSpace(resp.Choices[0].Message.Content)
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	sql = strings.TrimSpace(sql)

	return &SmartQueryResponse{
		SQL: sql,
		Request: QueryRequest{
			// AI queries might be complex, so we might not be able to reconstruct the full structured request object easily.
			// For now, we return empty structured request or partial.
			// The frontend should rely on the SQL for execution.
			Page:  1,
			Limit: 100,
		},
	}, nil
}

func buildLegacySmartQuery(prompt string, now time.Time) (*SmartQueryResponse, error) {
	if containsUnsafeTokens(prompt) {
		return nil, fmt.Errorf("prompt contiene token non consentiti")
	}

	minutes := extractRangeMinutes(prompt)
	from := now.Add(-time.Duration(minutes) * time.Minute)
	to := now

	filters := map[string]string{}
	if serviceName := extractServiceName(prompt); serviceName != "" {
		filters["service.name"] = serviceName
	}
	if traceID := extractTraceID(prompt); traceID != "" {
		filters["trace_id"] = traceID
	}

	primary := detectPrimarySignal(prompt)
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
		return builders.BuildTracesQuery(filters, nil, from, to, limit, 0)
	case "metrics":
		return builders.BuildMetricsQuery(filters, nil, from, to, limit, 0)
	default:
		return builders.BuildLogsQuery(filters, nil, from, to, limit, 0)
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
