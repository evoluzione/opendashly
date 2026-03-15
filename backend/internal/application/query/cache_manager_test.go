package query

import (
	"sync"
	"testing"
	"time"
)

func TestCacheManager_ServicesCloneOnSetAndGet(t *testing.T) {
	c := newCacheManager(time.Second, time.Second)
	original := []string{"svc-a", "svc-b"}
	c.setServices(original)

	original[0] = "mutated"
	got, ok := c.getServices()
	if !ok {
		t.Fatal("expected services cache hit")
	}
	if got[0] != "svc-a" {
		t.Fatalf("expected cached value to be cloned, got %q", got[0])
	}

	got[1] = "changed"
	again, ok := c.getServices()
	if !ok {
		t.Fatal("expected services cache hit on second read")
	}
	if again[1] != "svc-b" {
		t.Fatalf("expected getServices to return clone, got %q", again[1])
	}
}

func TestCacheManager_ServicesTTLExpiration(t *testing.T) {
	c := newCacheManager(20*time.Millisecond, time.Second)
	c.setServices([]string{"svc-a"})

	if _, ok := c.getServices(); !ok {
		t.Fatal("expected services cache hit before ttl expires")
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.getServices(); ok {
		t.Fatal("expected services cache miss after ttl expires")
	}
}

func TestCacheManager_AttributesNormalizeKeyAndTTL(t *testing.T) {
	c := newCacheManager(time.Second, 20*time.Millisecond)
	c.setAttributes("  Service.Name  ", []string{"service.name"})

	got, ok := c.getAttributes("service.name")
	if !ok {
		t.Fatal("expected attributes cache hit with normalized key")
	}
	if len(got) != 1 || got[0] != "service.name" {
		t.Fatalf("unexpected cached attributes: %#v", got)
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := c.getAttributes("SERVICE.NAME"); ok {
		t.Fatal("expected attributes cache miss after ttl expires")
	}
}

func TestService_GetCacheManagerSingleton(t *testing.T) {
	svc := &Service{}
	const workers = 16

	results := make(chan *cacheManager, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			results <- svc.getCacheManager()
		}()
	}
	wg.Wait()
	close(results)

	var first *cacheManager
	for item := range results {
		if item == nil {
			t.Fatal("expected non-nil cache manager")
		}
		if first == nil {
			first = item
			continue
		}
		if item != first {
			t.Fatal("expected getCacheManager to return singleton instance")
		}
	}
}
