// Command otel-autotune-launcher sizes the OpenTelemetry collector's
// memory_limiter from the resources the container actually has, then execs the
// collector. It replaces the old small/standard/big machine profiles: there are
// no tiers to choose, the launcher reads the cgroup memory limit (or host RAM)
// at boot and derives a single safe, proportional configuration. The collector
// memory_limiter is fixed for the process lifetime, so unlike the backend this
// is a boot-time decision rather than a runtime feedback loop.
package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// fixed knobs that were identical across every old profile.
const (
	memoryLimiterCheckInterval = "1s"
	batchSendSize              = "500"
	batchTimeout               = "2s"
	exporterTimeout            = "10s"
	sendingQueueSize           = "5000"
	sendingQueueConsumers      = "1"
	retryInitialInterval       = "1s"
	retryMaxInterval           = "30s"
	retryMaxElapsedTime        = "0"
)

const (
	mib = 1024 * 1024

	// Floors keep the collector alive even on a tiny container.
	minMemoryLimitMiB = 64
	minSpikeLimitMiB  = 16

	// Fraction of the available memory the limiter is allowed to use, and the
	// spike (soft) headroom as a fraction of that limit.
	memoryLimitPercent = 75
	spikePercent       = 20

	// cgroup values at/above this are treated as "unlimited" sentinels.
	unlimitedThreshold = uint64(1) << 62
)

func main() {
	applyMemoryAutoTuning(realEnv{})
	execCollector()
}

// memorySizing is the derived collector memory configuration.
type memorySizing struct {
	limitMiB int
	spikeMiB int
}

// env abstracts the host so the sizing can be unit-tested.
type env interface {
	getenv(string) string
	readFile(string) ([]byte, error)
	hostRAMBytes() (uint64, bool)
}

func applyMemoryAutoTuning(e env) {
	sizing := resolveMemorySizing(e)
	mustSetEnv("OTEL_MEMORY_LIMITER_CHECK_INTERVAL", memoryLimiterCheckInterval)
	mustSetEnv("OTEL_MEMORY_LIMIT_MIB", strconv.Itoa(sizing.limitMiB))
	mustSetEnv("OTEL_MEMORY_SPIKE_LIMIT_MIB", strconv.Itoa(sizing.spikeMiB))
	mustSetEnv("OTEL_BATCH_SEND_SIZE", batchSendSize)
	mustSetEnv("OTEL_BATCH_TIMEOUT", batchTimeout)
	mustSetEnv("OTEL_EXPORTER_TIMEOUT", exporterTimeout)
	mustSetEnv("OTEL_SENDING_QUEUE_SIZE", sendingQueueSize)
	mustSetEnv("OTEL_SENDING_QUEUE_CONSUMERS", sendingQueueConsumers)
	mustSetEnv("OTEL_RETRY_INITIAL_INTERVAL", retryInitialInterval)
	mustSetEnv("OTEL_RETRY_MAX_INTERVAL", retryMaxInterval)
	mustSetEnv("OTEL_RETRY_MAX_ELAPSED_TIME", retryMaxElapsedTime)

	log.Printf("otel auto-tuning: memory_limit=%dMiB spike=%dMiB", sizing.limitMiB, sizing.spikeMiB)
}

// resolveMemorySizing derives the limiter sizing from, in order: an explicit
// OTEL_MEMORY_LIMIT_MIB override (escape hatch), the cgroup memory limit, or
// host RAM. A safe floor applies in all cases.
func resolveMemorySizing(e env) memorySizing {
	if override := strings.TrimSpace(e.getenv("OTEL_MEMORY_LIMIT_MIB")); override != "" {
		if mibVal, err := strconv.Atoi(override); err == nil && mibVal > 0 {
			return sizingFromLimitMiB(mibVal)
		}
	}

	availBytes, ok := detectMemoryLimitBytes(e)
	if !ok {
		// No signal at all: assume a modest container.
		return sizingFromAvailableMiB(256)
	}
	return sizingFromAvailableMiB(int(availBytes / mib))
}

// sizingFromAvailableMiB applies the proportional policy to total available MiB.
func sizingFromAvailableMiB(availMiB int) memorySizing {
	limit := availMiB * memoryLimitPercent / 100
	return sizingFromLimitMiB(limit)
}

// sizingFromLimitMiB clamps a chosen limit to the floor and derives the spike.
func sizingFromLimitMiB(limitMiB int) memorySizing {
	if limitMiB < minMemoryLimitMiB {
		limitMiB = minMemoryLimitMiB
	}
	spike := limitMiB * spikePercent / 100
	if spike < minSpikeLimitMiB {
		spike = minSpikeLimitMiB
	}
	return memorySizing{limitMiB: limitMiB, spikeMiB: spike}
}

// detectMemoryLimitBytes reads the cgroup (v2 then v1) memory limit, falling
// back to host RAM. Returns false only when nothing is readable.
func detectMemoryLimitBytes(e env) (uint64, bool) {
	if v, ok := readCgroupV2(e); ok {
		return v, true
	}
	if v, ok := readCgroupV1(e); ok {
		return v, true
	}
	return e.hostRAMBytes()
}

func readCgroupV2(e env) (uint64, bool) {
	data, err := e.readFile("/sys/fs/cgroup/memory.max")
	if err != nil {
		return 0, false
	}
	text := strings.TrimSpace(string(data))
	if text == "" || text == "max" {
		// Unlimited cgroup: defer to host RAM.
		return e.hostRAMBytes()
	}
	v, err := strconv.ParseUint(text, 10, 64)
	if err != nil || v == 0 || v >= unlimitedThreshold {
		return e.hostRAMBytes()
	}
	return v, true
}

func readCgroupV1(e env) (uint64, bool) {
	data, err := e.readFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || v == 0 || v >= unlimitedThreshold {
		return e.hostRAMBytes()
	}
	return v, true
}

func mustSetEnv(key string, value string) {
	if err := os.Setenv(key, value); err != nil {
		log.Fatalf("failed setting %s: %v", key, err)
	}
}

// realEnv is the production env backed by the OS.
type realEnv struct{}

func (realEnv) getenv(key string) string { return os.Getenv(key) }

func (realEnv) readFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (realEnv) hostRAMBytes() (uint64, bool) { return hostRAMBytes() }
