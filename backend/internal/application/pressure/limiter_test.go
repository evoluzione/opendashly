package pressure

import "testing"

func TestLimiterTryAcquireRejectsWhenSaturated(t *testing.T) {
	limiter := NewLimiter(1)

	release, ok := limiter.TryAcquire()
	if !ok {
		t.Fatal("expected first acquire to succeed")
	}
	defer release()

	if _, ok := limiter.TryAcquire(); ok {
		t.Fatal("expected saturated limiter to reject immediately")
	}
}

func TestLimiterDisabledWhenLimitIsZero(t *testing.T) {
	limiter := NewLimiter(0)

	release, ok := limiter.TryAcquire()
	if !ok {
		t.Fatal("expected zero limit limiter to behave as disabled")
	}
	release()
}
