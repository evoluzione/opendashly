package pressure

import (
	"errors"
	"sync"
)

var ErrBusy = errors.New("backend under telemetry pressure")

type ReleaseFunc func()

// Limiter is a concurrency gate whose limit can be resized at runtime by the
// adaptive tuning controller. A nil Limiter behaves as disabled (always admits).
type Limiter struct {
	mu    sync.Mutex
	count int
	limit int
}

func NewLimiter(limit int) *Limiter {
	if limit <= 0 {
		return nil
	}
	return &Limiter{limit: limit}
}

// SetLimit updates the maximum number of concurrent holders. Values below 1 are
// clamped to 1. Shrinking below the current in-flight count does not evict
// existing holders; it only blocks new acquisitions until they drain.
func (l *Limiter) SetLimit(n int) {
	if l == nil {
		return
	}
	if n < 1 {
		n = 1
	}
	l.mu.Lock()
	l.limit = n
	l.mu.Unlock()
}

// Limit returns the current maximum concurrency. Returns 0 for a disabled limiter.
func (l *Limiter) Limit() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

func (l *Limiter) TryAcquire() (ReleaseFunc, bool) {
	if l == nil {
		return func() {}, true
	}
	l.mu.Lock()
	if l.count >= l.limit {
		l.mu.Unlock()
		return nil, false
	}
	l.count++
	l.mu.Unlock()
	return l.release, true
}

func (l *Limiter) release() {
	l.mu.Lock()
	if l.count > 0 {
		l.count--
	}
	l.mu.Unlock()
}
