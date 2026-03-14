package query

import (
	"testing"
	"time"
)

func TestLogsCursor_RoundTrip(t *testing.T) {
	last := LogEntry{
		Timestamp: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
		TraceID:   "trace-1",
		SpanID:    "span-1",
	}

	encoded, err := encodeLogsCursor(last)
	if err != nil {
		t.Fatalf("encodeLogsCursor error: %v", err)
	}

	decoded, err := decodeLogsCursor(encoded)
	if err != nil {
		t.Fatalf("decodeLogsCursor error: %v", err)
	}
	if decoded == nil {
		t.Fatal("expected non-nil decoded cursor")
	}
	if !decoded.Timestamp.Equal(last.Timestamp) || decoded.TraceID != last.TraceID || decoded.SpanID != last.SpanID {
		t.Fatalf("unexpected decoded cursor: %#v", decoded)
	}
}

func TestTracesCursor_RoundTrip(t *testing.T) {
	last := TraceEntry{
		LastSeen: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
		TraceID:  "trace-2",
	}

	encoded, err := encodeTracesCursor(last)
	if err != nil {
		t.Fatalf("encodeTracesCursor error: %v", err)
	}

	decoded, err := decodeTracesCursor(encoded)
	if err != nil {
		t.Fatalf("decodeTracesCursor error: %v", err)
	}
	if decoded == nil {
		t.Fatal("expected non-nil decoded cursor")
	}
	if !decoded.LastSeen.Equal(last.LastSeen) || decoded.TraceID != last.TraceID {
		t.Fatalf("unexpected decoded cursor: %#v", decoded)
	}
}

func TestDecodeCursor_EmptyOrZeroValueReturnsNil(t *testing.T) {
	decodedLogs, err := decodeLogsCursor("")
	if err != nil {
		t.Fatalf("decodeLogsCursor empty error: %v", err)
	}
	if decodedLogs != nil {
		t.Fatalf("expected nil logs cursor for empty input, got %#v", decodedLogs)
	}

	encodedZeroLogs, err := encodeCursor(logsCursorPayload{})
	if err != nil {
		t.Fatalf("encode zero logs cursor error: %v", err)
	}
	decodedLogs, err = decodeLogsCursor(encodedZeroLogs)
	if err != nil {
		t.Fatalf("decode zero logs cursor error: %v", err)
	}
	if decodedLogs != nil {
		t.Fatalf("expected nil logs cursor for zero timestamp, got %#v", decodedLogs)
	}

	encodedZeroTraces, err := encodeCursor(tracesCursorPayload{})
	if err != nil {
		t.Fatalf("encode zero traces cursor error: %v", err)
	}
	decodedTraces, err := decodeTracesCursor(encodedZeroTraces)
	if err != nil {
		t.Fatalf("decode zero traces cursor error: %v", err)
	}
	if decodedTraces != nil {
		t.Fatalf("expected nil traces cursor for zero lastSeen, got %#v", decodedTraces)
	}
}
