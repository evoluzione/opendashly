package contract

import (
	"context"
	"errors"
	"sync"
	"time"

	"opendashly/backend/internal/auth"
)

var errNotFound = errors.New("not found")

type memoryRepo struct {
	mu     sync.Mutex
	users  map[string]auth.User
	byName map[string]string
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		users:  map[string]auth.User{},
		byName: map[string]string{},
	}
}

func (m *memoryRepo) Create(_ context.Context, user auth.User) (*auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if user.ID == "" {
		user.ID = "u-" + user.Username
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	m.users[user.ID] = user
	m.byName[user.Username] = user.ID
	return &user, nil
}

func (m *memoryRepo) GetByUsername(_ context.Context, username string) (*auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byName[username]
	if !ok {
		return nil, errNotFound
	}
	user := m.users[id]
	return &user, nil
}

func (m *memoryRepo) GetByID(_ context.Context, id string) (*auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return nil, errNotFound
	}
	return &user, nil
}

func (m *memoryRepo) List(_ context.Context) ([]auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	users := make([]auth.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *memoryRepo) UpdatePassword(_ context.Context, id, newHash string, mustChange bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	user.PasswordHash = newHash
	user.MustChangePassword = mustChange
	m.users[id] = user
	return nil
}

func (m *memoryRepo) UpdateLastLogin(_ context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	user.LastLoginAt = &at
	m.users[id] = user
	return nil
}
