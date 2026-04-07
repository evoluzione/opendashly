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

// NewClient creates a ClickHouse client from DSN.
func NewClient(ctx context.Context, dsn string, user string, password string) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{dsn},
		Auth: clickhouse.Auth{
			Username: user,
			Password: password,
		},
		MaxOpenConns:    5,
		MaxIdleConns:    3,
		ConnMaxLifetime: time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     10 * time.Second,
		Settings: clickhouse.Settings{
			"max_memory_usage":   200 * 1024 * 1024,
			"max_execution_time": 8,
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
