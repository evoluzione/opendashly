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
	"path/filepath"
	"strconv"
	"strings"
)

const (
	configTemplatePath = "/etc/otelcol/config.yaml"
	renderedConfigPath = "/tmp/otelcol-config.yaml"
	fileStorageDir     = "/var/lib/otelcol/queue"
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
	configPath := applyAutoTuning(realEnv{})
	execCollector(configPath)
}

// memorySizing is the derived collector memory configuration.
type memorySizing struct {
	limitMiB int
	spikeMiB int
}

// env abstracts the host so the sizing can be unit-tested.
type env interface {
	readFile(string) ([]byte, error)
	hostRAMBytes() (uint64, bool)
}

func applyAutoTuning(e env) string {
	sizing := resolveMemorySizing(e)
	values := map[string]string{
		"MEMORY_LIMITER_CHECK_INTERVAL": memoryLimiterCheckInterval,
		"MEMORY_LIMIT_MIB":              strconv.Itoa(sizing.limitMiB),
		"MEMORY_SPIKE_LIMIT_MIB":        strconv.Itoa(sizing.spikeMiB),
		"BATCH_SEND_SIZE":               batchSendSize,
		"BATCH_TIMEOUT":                 batchTimeout,
		"EXPORTER_TIMEOUT":              exporterTimeout,
		"SENDING_QUEUE_SIZE":            sendingQueueSize,
		"SENDING_QUEUE_CONSUMERS":       sendingQueueConsumers,
		"RETRY_INITIAL_INTERVAL":        retryInitialInterval,
		"RETRY_MAX_INTERVAL":            retryMaxInterval,
		"RETRY_MAX_ELAPSED_TIME":        retryMaxElapsedTime,
		"FILE_STORAGE_DIR":              fileStorageDir,
	}
	renderConfig(e, values)
	log.Printf("otel auto-tuning: memory_limit=%dMiB spike=%dMiB", sizing.limitMiB, sizing.spikeMiB)
	return renderedConfigPath
}

func renderConfig(e env, values map[string]string) {
	data, err := e.readFile(configTemplatePath)
	if err != nil {
		log.Fatalf("failed reading collector config template: %v", err)
	}
	out := string(data)
	for key, value := range values {
		out = strings.ReplaceAll(out, "__"+key+"__", value)
	}
	if strings.Contains(out, "__") {
		log.Fatalf("collector config template contains unresolved auto-tuning placeholders")
	}
	if err := os.MkdirAll(filepath.Dir(renderedConfigPath), 0o755); err != nil {
		log.Fatalf("failed creating rendered config directory: %v", err)
	}
	if err := os.WriteFile(renderedConfigPath, []byte(out), 0o600); err != nil {
		log.Fatalf("failed writing rendered collector config: %v", err)
	}
}

// resolveMemorySizing derives the limiter sizing from the cgroup memory limit
// or host RAM. A safe floor applies in all cases; external env overrides are no
// longer accepted because the collector is fully auto-tuned.
func resolveMemorySizing(e env) memorySizing {
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

// realEnv is the production host backed by the OS.
type realEnv struct{}

func (realEnv) readFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (realEnv) hostRAMBytes() (uint64, bool) { return hostRAMBytes() }
