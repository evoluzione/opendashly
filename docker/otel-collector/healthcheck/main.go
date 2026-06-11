// Command otel-healthcheck probes the collector's health_check extension and
// exits non-zero when it does not answer with a 2xx. The collector image is
// distroless (no shell, no wget), so Docker healthchecks need a dedicated
// static binary.
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	url := os.Getenv("OTEL_HEALTH_URL")
	if url == "" {
		url = "http://127.0.0.1:13133/"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		os.Exit(1)
	}
}
