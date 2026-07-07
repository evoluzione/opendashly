// Package tuning implements a runtime self-tuning controller that replaces the
// old static small/standard/big machine profiles. Instead of mapping a machine
// to a frozen table of knobs, the controller starts every deployment at a safe
// floor and adapts the load-sensitive knobs at runtime using an AIMD policy
// (additive-increase / multiplicative-decrease), the same congestion-control
// idea used by TCP: grow slowly while the backend is calm, cut sharply the
// moment it shows pressure. This keeps the system resilient on the weakest
// hardware while still using spare capacity on a larger host — with no
// configuration and no machine sizing.
package tuning

import (
	"context"
	"log"
	"sync"
	"time"

	"opendashly/backend/internal/application/pressure"
	"opendashly/backend/internal/infrastructure/config"
)

// Bounds defines the floors, ceilings and step sizes the controller operates
// within. Zero values are replaced by DefaultBounds.
type Bounds struct {
	ParallelismFloor int
	ParallelismCeil  int
	ConcurrencyFloor int
	ConcurrencyCeil  int
	CHMemFloorMiB    int
	CHMemCeilMiB     int
	CHMemStepMiB     int
	// Interval is how often the control loop samples pressure.
	Interval time.Duration
	// CalmWindow is how long the backend must stay calm before the controller
	// grows a knob by one additive step.
	CalmWindow time.Duration
}

// DefaultBounds is tuned to be safe on a ~1 vCPU / 1-2 GiB container while still
// being able to grow into a larger host.
func DefaultBounds() Bounds {
	auto := config.Auto()
	cpu := auto.Resources.CPUCores
	if cpu <= 0 {
		cpu = 1
	}
	mem := auto.Resources.MemoryMiB
	if mem <= 0 {
		mem = 1024
	}
	return Bounds{
		ParallelismFloor: 1,
		ParallelismCeil:  clampInt(cpu, 1, 8),
		ConcurrencyFloor: 1,
		ConcurrencyCeil:  clampInt(cpu*2, 1, 12),
		CHMemFloorMiB:    auto.ClickHouseMaxMemoryMiB,
		CHMemCeilMiB:     clampInt(mem/4, auto.ClickHouseMaxMemoryMiB, 512),
		CHMemStepMiB:     32,
		Interval:         time.Duration(auto.TuningIntervalSec) * time.Second,
		CalmWindow:       time.Duration(auto.TuningCalmWindowSec) * time.Second,
	}
}

func (b Bounds) withDefaults() Bounds {
	d := DefaultBounds()
	if b.ParallelismFloor <= 0 {
		b.ParallelismFloor = d.ParallelismFloor
	}
	if b.ParallelismCeil < b.ParallelismFloor {
		b.ParallelismCeil = max(d.ParallelismCeil, b.ParallelismFloor)
	}
	if b.ConcurrencyFloor <= 0 {
		b.ConcurrencyFloor = d.ConcurrencyFloor
	}
	if b.ConcurrencyCeil < b.ConcurrencyFloor {
		b.ConcurrencyCeil = max(d.ConcurrencyCeil, b.ConcurrencyFloor)
	}
	if b.CHMemFloorMiB <= 0 {
		b.CHMemFloorMiB = d.CHMemFloorMiB
	}
	if b.CHMemCeilMiB < b.CHMemFloorMiB {
		b.CHMemCeilMiB = max(d.CHMemCeilMiB, b.CHMemFloorMiB)
	}
	if b.CHMemStepMiB <= 0 {
		b.CHMemStepMiB = d.CHMemStepMiB
	}
	if b.Interval <= 0 {
		b.Interval = d.Interval
	}
	if b.CalmWindow <= 0 {
		b.CalmWindow = d.CalmWindow
	}
	return b
}

// Controller holds the current adaptive knob levels and the policy that moves
// them. It is safe for concurrent use.
type Controller struct {
	bounds  Bounds
	probe   func(context.Context) bool
	limiter *pressure.Limiter
	now     func() time.Time

	mu          sync.Mutex
	parallelism int
	concurrency int
	chMemMiB    int
	lastChange  time.Time
}

// New builds a controller starting at the configured floors. probe reports
// whether the backend is currently under pressure; limiter (optional) is kept
// in sync with the concurrency level so the telemetry gate resizes live.
func New(bounds Bounds, probe func(context.Context) bool, limiter *pressure.Limiter) *Controller {
	b := bounds.withDefaults()
	c := &Controller{
		bounds:      b,
		probe:       probe,
		limiter:     limiter,
		now:         time.Now,
		parallelism: b.ParallelismFloor,
		concurrency: b.ConcurrencyFloor,
		chMemMiB:    b.CHMemFloorMiB,
	}
	c.lastChange = c.now()
	c.syncLimiter()
	return c
}

// Parallelism returns the current dashboard query fan-out.
func (c *Controller) Parallelism() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.parallelism
}

// Concurrency returns the current telemetry concurrency limit.
func (c *Controller) Concurrency() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.concurrency
}

// CHMaxMemoryBytes returns the current per-query ClickHouse memory budget in bytes.
func (c *Controller) CHMaxMemoryBytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chMemMiB * 1024 * 1024
}

// Run drives the control loop until ctx is cancelled.
func (c *Controller) Run(ctx context.Context) {
	ticker := time.NewTicker(c.bounds.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.tick(ctx)
		}
	}
}

// tick performs a single AIMD step. It is separated from Run so the policy can
// be unit-tested deterministically with an injected clock and probe.
func (c *Controller) tick(ctx context.Context) {
	under := c.probe(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()

	if under {
		c.decreaseLocked()
		return
	}
	if c.now().Sub(c.lastChange) >= c.bounds.CalmWindow {
		c.increaseLocked()
	}
}

// decreaseLocked applies the multiplicative cut on pressure.
func (c *Controller) decreaseLocked() {
	np := max(c.bounds.ParallelismFloor, c.parallelism/2)
	nc := max(c.bounds.ConcurrencyFloor, c.concurrency/2)
	nm := max(c.bounds.CHMemFloorMiB, c.chMemMiB*7/10)
	c.applyLocked(np, nc, nm, "pressure")
}

// increaseLocked applies one additive step after sustained calm.
func (c *Controller) increaseLocked() {
	np := min(c.bounds.ParallelismCeil, c.parallelism+1)
	nc := min(c.bounds.ConcurrencyCeil, c.concurrency+1)
	nm := min(c.bounds.CHMemCeilMiB, c.chMemMiB+c.bounds.CHMemStepMiB)
	c.applyLocked(np, nc, nm, "calm")
}

func (c *Controller) applyLocked(np, nc, nm int, reason string) {
	if np == c.parallelism && nc == c.concurrency && nm == c.chMemMiB {
		return
	}
	log.Printf("tuning: parallelism %d->%d concurrency %d->%d ch_mem_mib %d->%d reason=%s",
		c.parallelism, np, c.concurrency, nc, c.chMemMiB, nm, reason)
	c.parallelism = np
	c.concurrency = nc
	c.chMemMiB = nm
	c.lastChange = c.now()
	c.syncLimiter()
}

func (c *Controller) syncLimiter() {
	c.limiter.SetLimit(c.concurrency)
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
