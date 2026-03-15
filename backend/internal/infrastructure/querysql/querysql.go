package querysql

import (
	"time"

	"opendashly/backend/internal/infrastructure/querysql/builders"
)

type FilterItem = builders.FilterItem

type LogsPageCursor = builders.LogsPageCursor

type TracesPageCursor = builders.TracesPageCursor

func BuildLogsQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int, cursor *LogsPageCursor) string {
	return builders.BuildLogsQuery(filters, filterList, from, to, limit, offset, cursor)
}

func BuildLogsCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	return builders.BuildLogsCountQuery(filters, filterList, from, to)
}

func BuildTracesQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int, cursor *TracesPageCursor) string {
	return builders.BuildTracesQuery(filters, filterList, from, to, limit, offset, cursor)
}

func BuildTracesCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	return builders.BuildTracesCountQuery(filters, filterList, from, to)
}

func BuildMetricsQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int) string {
	return builders.BuildMetricsQuery(filters, filterList, from, to, limit, offset)
}

func BuildMetricsCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	return builders.BuildMetricsCountQuery(filters, filterList, from, to)
}

func EscapeLiteral(value string) string {
	return builders.EscapeLiteral(value)
}

func EscapeTraceID(value string) string {
	return builders.EscapeTraceID(value)
}
