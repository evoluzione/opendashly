package query

import (
	"context"
	"fmt"
	"log"
	"sort"
)

const serviceNamesQuerySettings = `SETTINGS
	max_memory_usage = 67108864,
	max_bytes_before_external_group_by = 16777216,
	max_threads = 1,
	optimize_aggregation_in_order = 1`

var serviceRollupSourceTables = []string{
	"telemetry.dashboard_trace_service_1m",
	"telemetry.dashboard_log_level_1m",
	"telemetry.dashboard_trace_endpoint_1m",
}

var serviceRawSourceTables = []string{
	"telemetry.otel_logs",
	"telemetry.otel_traces",
}

// ListServices returns a sorted list of distinct service names.
func (s *Service) ListServices(ctx context.Context) ([]string, error) {
	if s.Storage == nil {
		return []string{}, nil
	}
	if services, ok := s.getServicesFromCache(); ok {
		return services, nil
	}

	services, err := s.listServiceNamesFromTables(ctx, serviceRollupSourceTables)
	if err != nil {
		log.Printf("services.list: rollup sources failed: %v", err)
	}
	if len(services) > 0 {
		s.setServicesCache(services)
		return services, nil
	}
	services = []string{}
	s.setServicesCache(services)
	return services, nil
}

func (s *Service) listServiceNamesFromTables(ctx context.Context, tables []string) ([]string, error) {
	seen := map[string]struct{}{}
	successes := 0

	for _, table := range tables {
		names, err := s.fetchServiceNames(ctx, table)
		if err != nil {
			log.Printf("services.list: %s failed: %v", table, err)
			continue
		}
		successes++
		for _, name := range names {
			seen[name] = struct{}{}
		}
	}
	if successes == 0 {
		return nil, fmt.Errorf("list services: all source tables failed")
	}
	services := make([]string, 0, len(seen))
	for name := range seen {
		services = append(services, name)
	}
	sort.Strings(services)
	return services, nil
}

func (s *Service) fetchServiceNames(ctx context.Context, table string) ([]string, error) {
	if s.serviceNameFetcher != nil {
		return s.serviceNameFetcher(ctx, table)
	}
	return s.queryServiceNames(ctx, table)
}

func (s *Service) queryServiceNames(ctx context.Context, table string) ([]string, error) {
	query := fmt.Sprintf(`SELECT ServiceName
	FROM %s
	WHERE ServiceName != ''
	GROUP BY ServiceName
	%s`, table, serviceNamesQuerySettings)
	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return names, nil
}
