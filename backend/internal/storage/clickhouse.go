package storage

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Client wraps ClickHouse connection for shared use.
type Client struct {
	Conn driver.Conn
}

// NewClient creates a ClickHouse client from DSN.
func NewClient(ctx context.Context, dsn string) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{dsn},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return &Client{Conn: conn}, nil
}
