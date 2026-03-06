package query

import "testing"

func TestRequestedSignals_DefaultAllWhenEmpty(t *testing.T) {
	got := requestedSignals(nil)
	if !got["logs"] || !got["traces"] || !got["metrics"] {
		t.Fatalf("expected all default signals enabled, got %#v", got)
	}
}

func TestRequestedSignals_UsesOnlyProvided(t *testing.T) {
	got := requestedSignals([]string{"logs", "metrics"})
	if !got["logs"] || !got["metrics"] {
		t.Fatalf("expected logs and metrics enabled, got %#v", got)
	}
	if got["traces"] {
		t.Fatalf("expected traces disabled, got %#v", got)
	}
}

func TestDecodeLogsCursor_InvalidPayload(t *testing.T) {
	if _, err := decodeLogsCursor("invalid-%%"); err == nil {
		t.Fatal("expected decodeLogsCursor to fail for invalid cursor")
	}
}

func TestDecodeTracesCursor_InvalidPayload(t *testing.T) {
	if _, err := decodeTracesCursor("invalid-%%"); err == nil {
		t.Fatal("expected decodeTracesCursor to fail for invalid cursor")
	}
}
