package query

import "fmt"

func determineQueryRunStatus(signals map[string]bool, signalErrors map[string]string) (string, error) {
	requestedCount := countRequestedSignals(signals)
	if len(signalErrors) == requestedCount && requestedCount > 0 {
		return "", fmt.Errorf("all requested signals failed: %s", joinSignalErrors(signalErrors))
	}
	if len(signalErrors) > 0 {
		return "partial", nil
	}
	return "complete", nil
}
