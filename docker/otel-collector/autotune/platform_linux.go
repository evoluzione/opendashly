//go:build linux

package main

import (
	"log"
	"os"
	"syscall"
)

// execCollector replaces this process with the collector, preserving the
// environment we just tuned.
func execCollector() {
	target := "/otelcol-contrib"
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"--config=/etc/otelcol/config.yaml"}
	}

	if err := syscall.Exec(target, append([]string{target}, args...), os.Environ()); err != nil {
		log.Fatalf("failed to exec otel collector: %v", err)
	}
}

// hostRAMBytes returns total physical RAM via sysinfo(2).
func hostRAMBytes() (uint64, bool) {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, false
	}
	unit := uint64(info.Unit)
	if unit == 0 {
		unit = 1
	}
	return uint64(info.Totalram) * unit, true
}
