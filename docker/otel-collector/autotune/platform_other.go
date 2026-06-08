//go:build !linux

package main

import "log"

// execCollector is only meaningful inside the Linux collector image. The
// non-Linux build exists solely so the sizing logic can be unit-tested on a
// developer machine.
func execCollector() {
	log.Fatal("otel-autotune-launcher is only supported on linux")
}

// hostRAMBytes has no portable implementation off Linux; callers fall back to a
// modest default when this returns false.
func hostRAMBytes() (uint64, bool) { return 0, false }
