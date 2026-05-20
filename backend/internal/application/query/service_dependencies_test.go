package query

import (
	"context"
	"reflect"
	"testing"

	"opendashly/backend/internal/infrastructure/storage"
)

func TestService_GetSignalRunner_DefaultIsLazySingleton(t *testing.T) {
	svc := &Service{}

	first := svc.getSignalRunner()
	if first == nil {
		t.Fatal("expected non-nil default signal runner")
	}

	second := svc.getSignalRunner()
	if second != first {
		t.Fatal("expected getSignalRunner to return singleton instance")
	}
}

func TestService_GetSignalRunner_UsesInjectedRunner(t *testing.T) {
	fake := &fakeSignalRunner{}
	svc := &Service{signalRunner: fake}

	if got := svc.getSignalRunner(); got != fake {
		t.Fatal("expected injected signal runner to be returned")
	}
}

func TestListServices_PrefersRollupTables(t *testing.T) {
	called := []string{}
	svc := &Service{
		Storage: &storage.Client{},
		serviceNameFetcher: func(_ context.Context, table string) ([]string, error) {
			called = append(called, table)
			switch table {
			case serviceRollupSourceTables[0]:
				return []string{"checkout", "api"}, nil
			case serviceRollupSourceTables[1]:
				return []string{"api"}, nil
			case serviceRollupSourceTables[2]:
				return []string{}, nil
			default:
				t.Fatalf("raw table %s should not be queried when rollups have services", table)
				return nil, nil
			}
		},
	}

	services, err := svc.ListServices(context.Background())
	if err != nil {
		t.Fatalf("ListServices returned error: %v", err)
	}
	if want := []string{"api", "checkout"}; !reflect.DeepEqual(services, want) {
		t.Fatalf("services = %#v, want %#v", services, want)
	}
	if want := len(serviceRollupSourceTables); len(called) != want {
		t.Fatalf("queried %d tables, want %d rollup tables", len(called), want)
	}
}

func TestListServices_ReturnsEmptyWhenRollupsAreEmptyWithoutRawFallback(t *testing.T) {
	called := []string{}
	svc := &Service{
		Storage: &storage.Client{},
		serviceNameFetcher: func(_ context.Context, table string) ([]string, error) {
			called = append(called, table)
			for _, rollupTable := range serviceRollupSourceTables {
				if table == rollupTable {
					return []string{}, nil
				}
			}
			t.Fatalf("raw table %s should not be queried when rollups are empty", table)
			return nil, nil
		},
	}

	services, err := svc.ListServices(context.Background())
	if err != nil {
		t.Fatalf("ListServices returned error: %v", err)
	}
	if want := []string{}; !reflect.DeepEqual(services, want) {
		t.Fatalf("services = %#v, want %#v", services, want)
	}
	if want := len(serviceRollupSourceTables); len(called) != want {
		t.Fatalf("queried %d tables, want %d rollup tables", len(called), want)
	}
}
