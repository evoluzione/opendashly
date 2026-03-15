package auth

import (
	"context"
	"time"
)

type User struct {
	ID                 string
	Username           string
	PasswordHash       string
	Role               string
	MustChangePassword bool
	IsDisabled         bool
	CreatedAt          time.Time
	LastLoginAt        *time.Time
}

type Repository interface {
	Create(ctx context.Context, user User) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]User, error)
	UpdatePassword(ctx context.Context, id, newHash string, mustChange bool) error
	UpdateLastLogin(ctx context.Context, id string, at time.Time) error
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)
