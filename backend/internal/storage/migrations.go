package storage

import (
	"context"
	"embed"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// isAlterTableStatement checks if the SQL statement is an ALTER TABLE command
var alterTableRegex = regexp.MustCompile(`(?i)ALTER\s+TABLE`)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ApplyMigrations executes bundled SQL migrations in ClickHouse.
func ApplyMigrations(ctx context.Context, conn driver.Conn) error {
	if err := conn.Exec(ctx, "CREATE DATABASE IF NOT EXISTS telemetry"); err != nil {
		return fmt.Errorf("ensure telemetry database: %w", err)
	}
	if err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS telemetry.schema_migrations (
		name String,
		applied_at DateTime
	) ENGINE = MergeTree()
	ORDER BY (name)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	applied := make(map[string]struct{})
	rows, err := conn.Query(ctx, "SELECT name FROM telemetry.schema_migrations")
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan applied migration: %w", err)
		}
		applied[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read applied migrations: %w", err)
	}
	rows.Close()

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
		if _, ok := applied[file]; ok {
			log.Printf("migration already applied: %s", file)
			continue
		}
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
				// ALTER TABLE statements may fail if referenced table doesn't exist yet
				// (e.g., tables created by the OTel collector). Treat as warning, not error.
				if alterTableRegex.MatchString(stmt) && strings.Contains(err.Error(), "Could not find table") {
					log.Printf("migration %s: skipping ALTER TABLE (table not yet created by collector): %s", file, err.Error())
					continue
				}
				return fmt.Errorf("apply migration %s statement %q: %w", file, stmt, err)
			}
			log.Printf("migration applied: %s", stmt)
		}
		if err := conn.Exec(ctx, "INSERT INTO telemetry.schema_migrations (name, applied_at) VALUES (?, ?)", file, time.Now()); err != nil {
			return fmt.Errorf("record migration %s: %w", file, err)
		}
	}
	return nil
}
