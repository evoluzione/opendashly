package retention

import "testing"

func TestEffectiveRetentionDays_AppliesStepDownAndClamp(t *testing.T) {
	svc := &CleanupService{
		AdaptiveOptions: AdaptiveRetentionOptions{
			StepDownDays:          2,
			MinTraceRetentionDays: 1,
			MinLogRetentionDays:   1,
		},
	}

	logs := svc.effectiveRetentionDays(RetentionSetting{SignalType: "logs", RetentionDays: 7}, 1)
	if logs != 5 {
		t.Fatalf("logs effective retention=%d, want 5", logs)
	}

	traces := svc.effectiveRetentionDays(RetentionSetting{SignalType: "traces", RetentionDays: 2}, 2)
	if traces != 1 {
		t.Fatalf("traces effective retention=%d, want 1", traces)
	}
}

func TestModeForLevel(t *testing.T) {
	if got := modeForLevel(0); got != RetentionModeNormal {
		t.Fatalf("modeForLevel(0)=%q, want %q", got, RetentionModeNormal)
	}
	if got := modeForLevel(1); got != RetentionModeReduced {
		t.Fatalf("modeForLevel(1)=%q, want %q", got, RetentionModeReduced)
	}
	if got := modeForLevel(2); got != RetentionModeEmergency {
		t.Fatalf("modeForLevel(2)=%q, want %q", got, RetentionModeEmergency)
	}
}
