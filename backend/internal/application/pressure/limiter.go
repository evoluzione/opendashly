package pressure

import "errors"

var ErrBusy = errors.New("backend under telemetry pressure")

type ReleaseFunc func()

type Limiter struct {
	sem chan struct{}
}

func NewLimiter(limit int) *Limiter {
	if limit <= 0 {
		return nil
	}
	return &Limiter{sem: make(chan struct{}, limit)}
}

func (l *Limiter) TryAcquire() (ReleaseFunc, bool) {
	if l == nil {
		return func() {}, true
	}
	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, true
	default:
		return nil, false
	}
}
