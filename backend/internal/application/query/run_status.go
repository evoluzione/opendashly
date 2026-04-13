package query

import (
	"fmt"
	"strings"
)

func determineQueryRunStatus(signals map[string]bool, signalErrors map[string]string) (string, error) {
	requestedCount := countRequestedSignals(signals)
	if len(signalErrors) == requestedCount && requestedCount > 0 {
		if areRecoverableSignalErrors(signalErrors) {
			return "partial", nil
		}
		return "", fmt.Errorf("all requested signals failed: %s", joinSignalErrors(signalErrors))
	}
	if len(signalErrors) > 0 {
		return "partial", nil
	}
	return "complete", nil
}

func areRecoverableSignalErrors(signalErrors map[string]string) bool {
	if len(signalErrors) == 0 {
		return false
	}
	for _, msg := range signalErrors {
		if !isRecoverableSignalError(msg) {
			return false
		}
	}
	return true
}

func isRecoverableSignalError(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	patterns := []string{
		"memory limit exceeded",
		"overcommittracker",
		"deadline exceeded",
		"timeout",
		"temporarily unavailable",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
