package storage

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ApplyMigrations executes bundled SQL migrations in ClickHouse.
func ApplyMigrations(ctx context.Context, conn driver.Conn) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	for _, file := range files {
		log.Printf("migration file start: %s", file)
		contents, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		statements := strings.Split(string(contents), ";")
		for _, statement := range statements {
			stmt := strings.TrimSpace(statement)
			if stmt == "" {
				continue
			}
			if err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("apply migration %s statement %q: %w", file, stmt, err)
			}
			log.Printf("migration applied: %s", stmt)
		}
	}
	return nil
}
