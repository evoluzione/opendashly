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

func TestLimiterSetLimitGrowsCapacity(t *testing.T) {
	limiter := NewLimiter(1)

	release1, ok := limiter.TryAcquire()
	if !ok {
		t.Fatal("expected first acquire to succeed")
	}
	defer release1()

	if _, ok := limiter.TryAcquire(); ok {
		t.Fatal("expected saturated limiter to reject before resize")
	}

	limiter.SetLimit(2)
	release2, ok := limiter.TryAcquire()
	if !ok {
		t.Fatal("expected acquire to succeed after growing the limit")
	}
	defer release2()
}

func TestLimiterSetLimitShrinkBlocksNewAcquires(t *testing.T) {
	limiter := NewLimiter(3)

	release, ok := limiter.TryAcquire()
	if !ok {
		t.Fatal("expected first acquire to succeed")
	}
	defer release()

	// Shrinking below the in-flight count must not evict the holder, but should
	// reject new acquisitions until capacity frees up.
	limiter.SetLimit(1)
	if _, ok := limiter.TryAcquire(); ok {
		t.Fatal("expected shrunk limiter to reject new acquires")
	}
}

func TestLimiterSetLimitClampsToOne(t *testing.T) {
	limiter := NewLimiter(2)
	limiter.SetLimit(0)
	if got := limiter.Limit(); got != 1 {
		t.Fatalf("Limit() = %d, want 1", got)
	}
}
