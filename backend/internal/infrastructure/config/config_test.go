package config

import "testing"

func TestResolveMachineProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		ramGB    int
		cpuCores int
		want     string
	}{
		{name: "explicit preset wins", profile: "standard", ramGB: 2, cpuCores: 1, want: "standard"},
		{name: "small by machine hints", ramGB: 2, cpuCores: 1, want: "small"},
		{name: "standard by machine hints", ramGB: 4, cpuCores: 2, want: "standard"},
		{name: "big by machine hints", ramGB: 8, cpuCores: 4, want: "big"},
		{name: "safest class when mismatched", ramGB: 8, cpuCores: 1, want: "small"},
		{name: "defaults to standard without hints", want: "standard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveMachineProfile(tt.profile, tt.ramGB, tt.cpuCores)
			if got != tt.want {
				t.Fatalf("profile = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyMachineTuningOverridesLegacyValues(t *testing.T) {
	cfg := &Config{
		MachineProfile:             "big",
		ClickHouseMaxMemoryMiB:     1,
		DashboardQueryParallelism:  99,
		ServiceListTimeoutSec:      99,
		CleanupIntervalMinutes:     99,
		DashboardRequestTimeoutSec: 99,
	}

	cfg.applyMachineTuning()

	if !cfg.MachineAutoTuning {
		t.Fatalf("MachineAutoTuning should be true")
	}
	if cfg.ResolvedMachineProfile != "big" {
		t.Fatalf("ResolvedMachineProfile = %q, want %q", cfg.ResolvedMachineProfile, "big")
	}
	if cfg.ClickHouseMaxMemoryMiB != 256 {
		t.Fatalf("ClickHouseMaxMemoryMiB = %d, want 256", cfg.ClickHouseMaxMemoryMiB)
	}
	if cfg.DashboardQueryParallelism != 2 {
		t.Fatalf("DashboardQueryParallelism = %d, want 2", cfg.DashboardQueryParallelism)
	}
	if cfg.ServiceListTimeoutSec != 15 {
		t.Fatalf("ServiceListTimeoutSec = %d, want 15", cfg.ServiceListTimeoutSec)
	}
	if cfg.CleanupIntervalMinutes != 360 {
		t.Fatalf("CleanupIntervalMinutes = %d, want 360", cfg.CleanupIntervalMinutes)
	}
	if cfg.MaxLogRetentionDays != 30 {
		t.Fatalf("MaxLogRetentionDays = %d, want 30", cfg.MaxLogRetentionDays)
	}
	if cfg.MaxTraceRetentionDays != 15 {
		t.Fatalf("MaxTraceRetentionDays = %d, want 15", cfg.MaxTraceRetentionDays)
	}
	if cfg.DashboardRequestTimeoutSec != 15 {
		t.Fatalf("DashboardRequestTimeoutSec = %d, want 15", cfg.DashboardRequestTimeoutSec)
	}
}
