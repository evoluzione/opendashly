package query

import (
	"context"
	"fmt"
	"sort"
)

// ListServices returns a sorted list of distinct service names.
func (s *Service) ListServices(ctx context.Context) ([]string, error) {
	if s.Storage == nil {
		return []string{}, nil
	}
	if services, ok := s.getServicesFromCache(); ok {
		return services, nil
	}
	query := `SELECT DISTINCT ServiceName FROM (
		SELECT ServiceName FROM telemetry.otel_logs
		UNION ALL
		SELECT ServiceName FROM telemetry.otel_traces
		UNION ALL
		SELECT ServiceName FROM telemetry.otel_metrics_sum
		UNION ALL
		SELECT ServiceName FROM telemetry.otel_metrics_gauge
	) WHERE ServiceName != ''`
	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()
	services := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		if name != "" {
			services = append(services, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate services: %w", err)
	}
	sort.Strings(services)
	s.setServicesCache(services)
	return services, nil
}
