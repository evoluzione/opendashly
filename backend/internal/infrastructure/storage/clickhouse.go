package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Client wraps ClickHouse connection for shared use.
type Client struct {
	Conn driver.Conn
}

type ClientOptions struct {
	DSN                           string
	User                          string
	Password                      string
	MaxOpenConns                  int
	MaxIdleConns                  int
	ConnMaxLifetime               time.Duration
	DialTimeout                   time.Duration
	ReadTimeout                   time.Duration
	MaxMemoryUsageBytes           int
	MaxBytesBeforeExternalGroupBy int
	MaxBytesBeforeExternalSort    int
	MaxTempDataOnDiskBytes        int
	MaxExecutionTimeSec           int
	MaxThreads                    int
}

// NewClient creates a ClickHouse client from DSN.
func NewClient(ctx context.Context, dsn string, user string, password string) (*Client, error) {
	return NewClientWithOptions(ctx, ClientOptions{
		DSN:                           dsn,
		User:                          user,
		Password:                      password,
		MaxOpenConns:                  5,
		MaxIdleConns:                  3,
		ConnMaxLifetime:               time.Hour,
		DialTimeout:                   5 * time.Second,
		ReadTimeout:                   10 * time.Second,
		MaxMemoryUsageBytes:           200 * 1024 * 1024,
		MaxBytesBeforeExternalGroupBy: 64 * 1024 * 1024,
		MaxBytesBeforeExternalSort:    64 * 1024 * 1024,
		MaxTempDataOnDiskBytes:        1024 * 1024 * 1024,
		MaxExecutionTimeSec:           8,
		MaxThreads:                    2,
	})
}

func NewClientWithOptions(ctx context.Context, opts ClientOptions) (*Client, error) {
	if opts.MaxOpenConns <= 0 {
		opts.MaxOpenConns = 5
	}
	if opts.MaxIdleConns <= 0 {
		opts.MaxIdleConns = 3
	}
	if opts.ConnMaxLifetime <= 0 {
		opts.ConnMaxLifetime = time.Hour
	}
	if opts.DialTimeout <= 0 {
		opts.DialTimeout = 5 * time.Second
	}
	if opts.ReadTimeout <= 0 {
		opts.ReadTimeout = 10 * time.Second
	}
	if opts.MaxMemoryUsageBytes <= 0 {
		opts.MaxMemoryUsageBytes = 200 * 1024 * 1024
	}
	if opts.MaxBytesBeforeExternalGroupBy <= 0 {
		opts.MaxBytesBeforeExternalGroupBy = 64 * 1024 * 1024
	}
	if opts.MaxBytesBeforeExternalSort <= 0 {
		opts.MaxBytesBeforeExternalSort = 64 * 1024 * 1024
	}
	if opts.MaxTempDataOnDiskBytes <= 0 {
		opts.MaxTempDataOnDiskBytes = 1024 * 1024 * 1024
	}
	if opts.MaxExecutionTimeSec <= 0 {
		opts.MaxExecutionTimeSec = 8
	}
	if opts.MaxThreads <= 0 {
		opts.MaxThreads = 2
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{opts.DSN},
		Auth: clickhouse.Auth{
			Username: opts.User,
			Password: opts.Password,
		},
		MaxOpenConns:    opts.MaxOpenConns,
		MaxIdleConns:    opts.MaxIdleConns,
		ConnMaxLifetime: opts.ConnMaxLifetime,
		DialTimeout:     opts.DialTimeout,
		ReadTimeout:     opts.ReadTimeout,
		Settings: clickhouse.Settings{
			"max_memory_usage":                          opts.MaxMemoryUsageBytes,
			"max_bytes_before_external_group_by":        opts.MaxBytesBeforeExternalGroupBy,
			"max_bytes_before_external_sort":            opts.MaxBytesBeforeExternalSort,
			"max_temporary_data_on_disk_size_for_query": opts.MaxTempDataOnDiskBytes,
			"max_execution_time":                        opts.MaxExecutionTimeSec,
			"max_threads":                               opts.MaxThreads,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return &Client{Conn: conn}, nil
}
