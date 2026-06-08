package tuning

import (
	"context"
	"testing"
	"time"

	"opendashly/backend/internal/application/pressure"
)

func testBounds() Bounds {
	return Bounds{
		ParallelismFloor: 1,
		ParallelismCeil:  4,
		ConcurrencyFloor: 1,
		ConcurrencyCeil:  6,
		CHMemFloorMiB:    64,
		CHMemCeilMiB:     256,
		CHMemStepMiB:     32,
		Interval:         time.Second,
		CalmWindow:       60 * time.Second,
	}
}

// fakeClock lets tests advance time deterministically.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }
func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func newTestController(t *testing.T, clock *fakeClock, probe func(context.Context) bool, limiter *pressure.Limiter) *Controller {
	t.Helper()
	c := New(testBounds(), probe, limiter)
	c.now = clock.now
	c.lastChange = clock.now()
	return c
}

func TestControllerStartsAtFloor(t *testing.T) {
	c := New(testBounds(), func(context.Context) bool { return false }, nil)
	if c.Parallelism() != 1 || c.Concurrency() != 1 {
		t.Fatalf("expected floors, got parallelism=%d concurrency=%d", c.Parallelism(), c.Concurrency())
	}
	if c.CHMaxMemoryBytes() != 64*1024*1024 {
		t.Fatalf("expected 64MiB floor, got %d bytes", c.CHMaxMemoryBytes())
	}
}

func TestControllerGrowsAfterSustainedCalm(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	c := newTestController(t, clock, func(context.Context) bool { return false }, nil)

	// Before the calm window elapses, no growth.
	clock.advance(30 * time.Second)
	c.tick(context.Background())
	if c.Parallelism() != 1 {
		t.Fatalf("expected no growth before calm window, got %d", c.Parallelism())
	}

	// After the calm window, one additive step.
	clock.advance(31 * time.Second)
	c.tick(context.Background())
	if c.Parallelism() != 2 || c.Concurrency() != 2 || c.CHMaxMemoryBytes() != 96*1024*1024 {
		t.Fatalf("expected one step up, got parallelism=%d concurrency=%d mem=%d",
			c.Parallelism(), c.Concurrency(), c.CHMaxMemoryBytes())
	}
}

func TestControllerCutsHardOnPressure(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	under := false
	c := newTestController(t, clock, func(context.Context) bool { return under }, nil)

	// Grow a few steps while calm.
	for i := 0; i < 3; i++ {
		clock.advance(61 * time.Second)
		c.tick(context.Background())
	}
	if c.Parallelism() != 4 { // capped at ceil
		t.Fatalf("expected to reach parallelism ceil 4, got %d", c.Parallelism())
	}
	if c.Concurrency() != 4 {
		t.Fatalf("expected concurrency 4 after 3 steps, got %d", c.Concurrency())
	}

	// Pressure halves parallelism/concurrency and cuts memory by 30%.
	under = true
	memBefore := c.CHMaxMemoryBytes()
	c.tick(context.Background())
	if c.Parallelism() != 2 || c.Concurrency() != 2 {
		t.Fatalf("expected halving on pressure, got parallelism=%d concurrency=%d", c.Parallelism(), c.Concurrency())
	}
	if c.CHMaxMemoryBytes() >= memBefore {
		t.Fatalf("expected memory to drop on pressure, before=%d after=%d", memBefore, c.CHMaxMemoryBytes())
	}
}

func TestControllerClampsAtFloorUnderSustainedPressure(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	c := newTestController(t, clock, func(context.Context) bool { return true }, nil)

	for i := 0; i < 10; i++ {
		c.tick(context.Background())
	}
	if c.Parallelism() != 1 || c.Concurrency() != 1 {
		t.Fatalf("expected floors under sustained pressure, got parallelism=%d concurrency=%d", c.Parallelism(), c.Concurrency())
	}
	if c.CHMaxMemoryBytes() != 64*1024*1024 {
		t.Fatalf("expected memory floor, got %d bytes", c.CHMaxMemoryBytes())
	}
}

func TestControllerResizesLimiter(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	limiter := pressure.NewLimiter(1)
	c := newTestController(t, clock, func(context.Context) bool { return false }, limiter)

	clock.advance(61 * time.Second)
	c.tick(context.Background())

	if got := limiter.Limit(); got != c.Concurrency() {
		t.Fatalf("limiter limit = %d, want %d (concurrency)", got, c.Concurrency())
	}
	if limiter.Limit() != 2 {
		t.Fatalf("expected limiter to grow to 2, got %d", limiter.Limit())
	}
}

func TestControllerPressureTakesPriorityOverCalm(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	c := newTestController(t, clock, func(context.Context) bool { return true }, nil)

	// Grow first while pretending calm by toggling, then assert pressure wins
	// even when the calm window has elapsed.
	clock.advance(120 * time.Second)
	c.tick(context.Background())
	if c.Parallelism() != 1 {
		t.Fatalf("expected pressure to prevent growth, got %d", c.Parallelism())
	}
}
