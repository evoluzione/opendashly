package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type Repo struct {
	Conn driver.Conn
}

func (r *Repo) Create(ctx context.Context, user User) (*User, error) {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	query := `INSERT INTO telemetry.users (id, username, password_hash, role, must_change_password, is_disabled, created_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	mustChange := boolToUInt8(user.MustChangePassword)
	disabled := boolToUInt8(user.IsDisabled)
	if err := r.Conn.Exec(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.Role,
		mustChange,
		disabled,
		user.CreatedAt,
		user.LastLoginAt,
	); err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &user, nil
}

func (r *Repo) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, password_hash, role, must_change_password, is_disabled, created_at, last_login_at
		FROM telemetry.users WHERE username = ? LIMIT 1`
	row := r.Conn.QueryRow(ctx, query, username)
	var user User
	var mustChange uint8
	var disabled uint8
	var lastLogin *time.Time
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &mustChange, &disabled, &user.CreatedAt, &lastLogin); err != nil {
		return nil, err
	}
	user.MustChangePassword = mustChange == 1
	user.IsDisabled = disabled == 1
	user.LastLoginAt = lastLogin
	return &user, nil
}

func (r *Repo) GetByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, username, password_hash, role, must_change_password, is_disabled, created_at, last_login_at
		FROM telemetry.users WHERE id = ? LIMIT 1`
	row := r.Conn.QueryRow(ctx, query, id)
	var user User
	var mustChange uint8
	var disabled uint8
	var lastLogin *time.Time
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &mustChange, &disabled, &user.CreatedAt, &lastLogin); err != nil {
		return nil, err
	}
	user.MustChangePassword = mustChange == 1
	user.IsDisabled = disabled == 1
	user.LastLoginAt = lastLogin
	return &user, nil
}

func (r *Repo) List(ctx context.Context) ([]User, error) {
	query := `SELECT id, username, password_hash, role, must_change_password, is_disabled, created_at, last_login_at
		FROM telemetry.users ORDER BY username ASC`
	rows, err := r.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var user User
		var mustChange uint8
		var disabled uint8
		var lastLogin *time.Time
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &mustChange, &disabled, &user.CreatedAt, &lastLogin); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.MustChangePassword = mustChange == 1
		user.IsDisabled = disabled == 1
		user.LastLoginAt = lastLogin
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

func (r *Repo) UpdatePassword(ctx context.Context, id, newHash string, mustChange bool) error {
	query := `ALTER TABLE telemetry.users UPDATE password_hash = ?, must_change_password = ? WHERE id = ?`
	if err := r.Conn.Exec(ctx, query, newHash, boolToUInt8(mustChange), id); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *Repo) UpdateLastLogin(ctx context.Context, id string, at time.Time) error {
	query := `ALTER TABLE telemetry.users UPDATE last_login_at = ? WHERE id = ?`
	if err := r.Conn.Exec(ctx, query, at, id); err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

func boolToUInt8(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}
