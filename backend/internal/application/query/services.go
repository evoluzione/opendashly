package query

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"
)

var serviceSourceTables = []string{
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

	var (
		mu        sync.Mutex
		seen      = map[string]struct{}{}
		successes int
	)

	g, gctx := errgroup.WithContext(ctx)
	for _, table := range serviceSourceTables {
		table := table
		g.Go(func() error {
			names, err := s.queryServiceNames(gctx, table)
			if err != nil {
				log.Printf("services.list: %s failed: %v", table, err)
				return nil
			}
			mu.Lock()
			successes++
			for _, name := range names {
				seen[name] = struct{}{}
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if successes == 0 {
		return nil, fmt.Errorf("list services: all source tables failed")
	}

	services := make([]string, 0, len(seen))
	for name := range seen {
		services = append(services, name)
	}
	sort.Strings(services)
	s.setServicesCache(services)
	return services, nil
}

func (s *Service) queryServiceNames(ctx context.Context, table string) ([]string, error) {
	query := fmt.Sprintf(`SELECT ServiceName
FROM %s
WHERE ServiceName != ''
GROUP BY ServiceName
SETTINGS optimize_aggregation_in_order = 1`, table)
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
