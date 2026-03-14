package query

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"opendashly/backend/internal/infrastructure/querysql"
)

type logsCursorPayload struct {
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"traceId"`
	SpanID    string    `json:"spanId"`
}

type tracesCursorPayload struct {
	LastSeen time.Time `json:"lastSeen"`
	TraceID  string    `json:"traceId"`
}

func decodeLogsCursor(value string) (*querysql.LogsPageCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var payload logsCursorPayload
	if err := decodeCursor(value, &payload); err != nil {
		return nil, err
	}
	if payload.Timestamp.IsZero() {
		return nil, nil
	}
	return &querysql.LogsPageCursor{
		Timestamp: payload.Timestamp,
		TraceID:   payload.TraceID,
		SpanID:    payload.SpanID,
	}, nil
}

func decodeTracesCursor(value string) (*querysql.TracesPageCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var payload tracesCursorPayload
	if err := decodeCursor(value, &payload); err != nil {
		return nil, err
	}
	if payload.LastSeen.IsZero() {
		return nil, nil
	}
	return &querysql.TracesPageCursor{
		LastSeen: payload.LastSeen,
		TraceID:  payload.TraceID,
	}, nil
}

func encodeLogsCursor(last LogEntry) (string, error) {
	return encodeCursor(logsCursorPayload{
		Timestamp: last.Timestamp,
		TraceID:   last.TraceID,
		SpanID:    last.SpanID,
	})
}

func encodeTracesCursor(last TraceEntry) (string, error) {
	return encodeCursor(tracesCursorPayload{
		LastSeen: last.LastSeen,
		TraceID:  last.TraceID,
	})
}

func encodeCursor(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(encoded string, dest any) error {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}
