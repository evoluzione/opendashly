package query

import (
	"context"
	"fmt"
	"strings"

	"opendashly/backend/internal/infrastructure/config"
	"opendashly/backend/internal/infrastructure/querysql"
)

var defaultLogAttributeSeed = []string{
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

func buildLogAttributeKeysQuery(search string) string {
	query := "SELECT DISTINCT key FROM (" +
		"SELECT arrayJoin(mapKeys(ResourceAttributes)) AS key FROM (" +
		"SELECT ResourceAttributes FROM telemetry.otel_logs " +
		"WHERE Timestamp >= now() - INTERVAL 24 HOUR " +
		"ORDER BY Timestamp DESC LIMIT 5000" +
		") " +
		"UNION ALL " +
		"SELECT arrayJoin(mapKeys(LogAttributes)) AS key FROM (" +
		"SELECT LogAttributes FROM telemetry.otel_logs " +
		"WHERE Timestamp >= now() - INTERVAL 24 HOUR " +
		"ORDER BY Timestamp DESC LIMIT 5000" +
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
		escaped := querysql.EscapeLiteral(search)
		query += " AND key ILIKE '%" + escaped + "%'"
	}
	query += " ORDER BY key LIMIT 100 SETTINGS max_execution_time = " + fmt.Sprintf("%d", config.Auto().LogAttributeKeysMaxExecSec) + ", max_threads = 1, max_memory_usage = 33554432, max_bytes_before_external_sort = 8388608"
	return query
}

func fallbackLogAttributeKeys(search string) []string {
	searchLower := strings.ToLower(strings.TrimSpace(search))
	filtered := make([]string, 0, len(defaultLogAttributeSeed))
	for _, key := range defaultLogAttributeSeed {
		if searchLower == "" || strings.Contains(strings.ToLower(key), searchLower) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}

func (s *Service) GetLogAttributeKeys(ctx context.Context, search string) ([]string, error) {
	if s.Storage == nil {
		return []string{}, nil
	}
	if keys, ok := s.getAttributesFromCache(search); ok {
		return keys, nil
	}
	release, ok := s.Limiter.TryAcquire()
	if !ok {
		filtered := fallbackLogAttributeKeys(search)
		s.setAttributesCache(search, filtered)
		return filtered, nil
	}
	defer release()

	query := buildLogAttributeKeysQuery(search)

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		filtered := fallbackLogAttributeKeys(search)
		s.setAttributesCache(search, filtered)
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "memory limit exceeded") ||
			strings.Contains(lower, "overcommittracker") ||
			strings.Contains(lower, "timeout") ||
			strings.Contains(lower, "deadline exceeded") {
			return filtered, nil
		}
		return filtered, fmt.Errorf("query attributes: %w", err)
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

	filtered := fallbackLogAttributeKeys(search)
	s.setAttributesCache(search, filtered)
	return filtered, nil
}
