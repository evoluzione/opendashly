package auth

import (
	"sync"
	"time"
)

type cachedUser struct {
	user      User
	expiresAt time.Time
}

type UserCache struct {
	ttl   time.Duration
	mu    sync.Mutex
	users map[string]cachedUser
}

func NewUserCache(ttl time.Duration) *UserCache {
	if ttl <= 0 {
		return nil
	}
	return &UserCache{
		ttl:   ttl,
		users: map[string]cachedUser{},
	}
}

func (c *UserCache) Get(id string) (User, bool) {
	if c == nil || id == "" {
		return User{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.users[id]
	if !ok {
		return User{}, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.users, id)
		return User{}, false
	}
	return cloneUser(entry.user), true
}

func (c *UserCache) Set(user User) {
	if c == nil || user.ID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users[user.ID] = cachedUser{
		user:      cloneUser(user),
		expiresAt: time.Now().Add(c.ttl),
	}
}

func cloneUser(user User) User {
	if user.LastLoginAt != nil {
		lastLogin := *user.LastLoginAt
		user.LastLoginAt = &lastLogin
	}
	return user
}
