//go:build linux

package main

import (
	"log"
	"os"
	"strings"
	"syscall"
)

// execCollector replaces this process with the collector using the rendered
// auto-tuned config.
func execCollector(configPath string) {
	target := "/otelcol-contrib"
	args := []string{"--config=" + configPath}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--config") {
			continue
		}
		args = append(args, arg)
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
