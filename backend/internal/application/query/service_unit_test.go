package query

import "testing"

func TestRequestedSignals_DefaultAllWhenEmpty(t *testing.T) {
	got := requestedSignals(nil)
	if !got["logs"] || !got["traces"] {
		t.Fatalf("expected default logs and traces enabled, got %#v", got)
	}
}

func TestRequestedSignals_UsesOnlyProvided(t *testing.T) {
	got := requestedSignals([]string{"logs", "traces"})
	if !got["logs"] || !got["traces"] {
		t.Fatalf("expected logs and traces enabled, got %#v", got)
	}
	if len(got) != 2 {
		t.Fatalf("expected only two signals, got %#v", got)
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
