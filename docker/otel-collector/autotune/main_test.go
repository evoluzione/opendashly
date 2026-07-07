package main

import "testing"

// fakeEnv drives the sizing logic deterministically across platforms.
type fakeEnv struct {
	files   map[string][]byte
	hostRAM uint64
	hasHost bool
}

func (e fakeEnv) readFile(p string) ([]byte, error) {
	if data, ok := e.files[p]; ok {
		return data, nil
	}
	return nil, errNotFound
}

func (e fakeEnv) hostRAMBytes() (uint64, bool) { return e.hostRAM, e.hasHost }

var errNotFound = &fileErr{}

type fileErr struct{}

func (*fileErr) Error() string { return "not found" }

func TestSizingFromCgroupV2(t *testing.T) {
	e := fakeEnv{
		files: map[string][]byte{
			"/sys/fs/cgroup/memory.max": []byte("1073741824\n"), // 1 GiB
		},
	}
	s := resolveMemorySizing(e)
	// 1024 MiB * 75% = 768; spike 768 * 20% = 153.
	if s.limitMiB != 768 {
		t.Fatalf("limitMiB = %d, want 768", s.limitMiB)
	}
	if s.spikeMiB != 153 {
		t.Fatalf("spikeMiB = %d, want 153", s.spikeMiB)
	}
}

func TestSizingCgroupV2MaxFallsBackToHost(t *testing.T) {
	e := fakeEnv{
		files:   map[string][]byte{"/sys/fs/cgroup/memory.max": []byte("max")},
		hostRAM: 2 * 1024 * 1024 * 1024, // 2 GiB
		hasHost: true,
	}
	s := resolveMemorySizing(e)
	// 2048 MiB * 75% = 1536.
	if s.limitMiB != 1536 {
		t.Fatalf("limitMiB = %d, want 1536", s.limitMiB)
	}
}

func TestSizingCgroupV1Sentinel(t *testing.T) {
	e := fakeEnv{
		files: map[string][]byte{
			// v1 "unlimited" sentinel must defer to host RAM.
			"/sys/fs/cgroup/memory/memory.limit_in_bytes": []byte("9223372036854771712"),
		},
		hostRAM: 1024 * 1024 * 1024, // 1 GiB
		hasHost: true,
	}
	s := resolveMemorySizing(e)
	if s.limitMiB != 768 {
		t.Fatalf("limitMiB = %d, want 768", s.limitMiB)
	}
}

func TestSizingAppliesFloorOnTinyLimit(t *testing.T) {
	e := fakeEnv{
		files: map[string][]byte{"/sys/fs/cgroup/memory.max": []byte("33554432")}, // 32 MiB
	}
	s := resolveMemorySizing(e)
	if s.limitMiB != minMemoryLimitMiB {
		t.Fatalf("limitMiB = %d, want floor %d", s.limitMiB, minMemoryLimitMiB)
	}
	if s.spikeMiB != minSpikeLimitMiB {
		t.Fatalf("spikeMiB = %d, want floor %d", s.spikeMiB, minSpikeLimitMiB)
	}
}

func TestSizingNoSignalUsesModestDefault(t *testing.T) {
	s := resolveMemorySizing(fakeEnv{})
	// 256 MiB * 75% = 192.
	if s.limitMiB != 192 {
		t.Fatalf("limitMiB = %d, want 192", s.limitMiB)
	}
}
