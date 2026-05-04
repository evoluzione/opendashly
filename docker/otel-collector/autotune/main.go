package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type otelPreset struct {
	memoryLimiterCheckInterval string
	memoryLimitMiB             string
	memorySpikeLimitMiB        string
	batchSendSize              string
	batchTimeout               string
	exporterTimeout            string
	sendingQueueSize           string
	sendingQueueConsumers      string
	retryInitialInterval       string
	retryMaxInterval           string
	retryMaxElapsedTime        string
}

var otelPresets = map[string]otelPreset{
	"small": {
		memoryLimiterCheckInterval: "1s",
		memoryLimitMiB:             "96",
		memorySpikeLimitMiB:        "20",
		batchSendSize:              "500",
		batchTimeout:               "2s",
		exporterTimeout:            "10s",
		sendingQueueSize:           "5000",
		sendingQueueConsumers:      "1",
		retryInitialInterval:       "1s",
		retryMaxInterval:           "30s",
		retryMaxElapsedTime:        "0",
	},
	"standard": {
		memoryLimiterCheckInterval: "1s",
		memoryLimitMiB:             "170",
		memorySpikeLimitMiB:        "35",
		batchSendSize:              "500",
		batchTimeout:               "2s",
		exporterTimeout:            "10s",
		sendingQueueSize:           "5000",
		sendingQueueConsumers:      "1",
		retryInitialInterval:       "1s",
		retryMaxInterval:           "30s",
		retryMaxElapsedTime:        "0",
	},
	"big": {
		memoryLimiterCheckInterval: "1s",
		memoryLimitMiB:             "300",
		memorySpikeLimitMiB:        "60",
		batchSendSize:              "500",
		batchTimeout:               "2s",
		exporterTimeout:            "10s",
		sendingQueueSize:           "5000",
		sendingQueueConsumers:      "1",
		retryInitialInterval:       "1s",
		retryMaxInterval:           "30s",
		retryMaxElapsedTime:        "0",
	},
}

func main() {
	applyMachineAutoTuning()
	execCollector()
}

func applyMachineAutoTuning() {
	profile := resolveMachineProfile(
		strings.ToLower(strings.TrimSpace(os.Getenv("MACHINE_PROFILE"))),
		getEnvInt("MACHINE_RAM_GB", 0),
		getEnvInt("MACHINE_CPU_CORES", 0),
	)

	preset, ok := otelPresets[profile]
	if !ok {
		preset = otelPresets["standard"]
		profile = "standard"
	}

	// Auto-tuning intentionally overrides OTEL_* knobs whenever MACHINE_* is provided.
	mustSetEnv("OTEL_MEMORY_LIMITER_CHECK_INTERVAL", preset.memoryLimiterCheckInterval)
	mustSetEnv("OTEL_MEMORY_LIMIT_MIB", preset.memoryLimitMiB)
	mustSetEnv("OTEL_MEMORY_SPIKE_LIMIT_MIB", preset.memorySpikeLimitMiB)
	mustSetEnv("OTEL_BATCH_SEND_SIZE", preset.batchSendSize)
	mustSetEnv("OTEL_BATCH_TIMEOUT", preset.batchTimeout)
	mustSetEnv("OTEL_EXPORTER_TIMEOUT", preset.exporterTimeout)
	mustSetEnv("OTEL_SENDING_QUEUE_SIZE", preset.sendingQueueSize)
	mustSetEnv("OTEL_SENDING_QUEUE_CONSUMERS", preset.sendingQueueConsumers)
	mustSetEnv("OTEL_RETRY_INITIAL_INTERVAL", preset.retryInitialInterval)
	mustSetEnv("OTEL_RETRY_MAX_INTERVAL", preset.retryMaxInterval)
	mustSetEnv("OTEL_RETRY_MAX_ELAPSED_TIME", preset.retryMaxElapsedTime)

	log.Printf("otel machine auto-tuning enabled: profile=%s", profile)
}

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

func resolveMachineProfile(profile string, ramGB int, cpuCores int) string {
	if _, ok := otelPresets[profile]; ok {
		return profile
	}

	if ramGB <= 0 && cpuCores <= 0 {
		return "standard"
	}

	classByRAM := resourceClassFromRAM(ramGB)
	classByCPU := resourceClassFromCPU(cpuCores)
	class := minPositiveClass(classByRAM, classByCPU)

	switch class {
	case 1:
		return "small"
	case 2:
		return "standard"
	default:
		return "big"
	}
}

func resourceClassFromRAM(ramGB int) int {
	if ramGB <= 0 {
		return 0
	}
	if ramGB <= 2 {
		return 1
	}
	if ramGB <= 4 {
		return 2
	}
	return 3
}

func resourceClassFromCPU(cpuCores int) int {
	if cpuCores <= 0 {
		return 0
	}
	if cpuCores <= 1 {
		return 1
	}
	if cpuCores <= 2 {
		return 2
	}
	return 3
}

func minPositiveClass(a int, b int) int {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func getEnvInt(key string, defaultVal int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func mustSetEnv(key string, value string) {
	if err := os.Setenv(key, value); err != nil {
		log.Fatalf("failed setting %s: %v", key, err)
	}
}
