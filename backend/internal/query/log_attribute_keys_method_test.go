package query

import (
	"context"
	"testing"

	"opendashly/backend/internal/infrastructure/storage"
)

func TestGetLogAttributeKeys_StorageNilReturnsEmpty(t *testing.T) {
	svc := &Service{}
	got, err := svc.GetLogAttributeKeys(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty keys, got %#v", got)
	}
}

func TestGetLogAttributeKeys_UsesCacheWithoutDBConn(t *testing.T) {
	svc := &Service{Storage: &storage.Client{}}
	svc.setAttributesCache("service", []string{"service.name"})

	got, err := svc.GetLogAttributeKeys(context.Background(), "service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "service.name" {
		t.Fatalf("unexpected cached keys: %#v", got)
	}
}
