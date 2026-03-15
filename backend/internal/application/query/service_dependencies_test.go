package query

import "testing"

func TestService_GetSignalRunner_DefaultIsLazySingleton(t *testing.T) {
	svc := &Service{}

	first := svc.getSignalRunner()
	if first == nil {
		t.Fatal("expected non-nil default signal runner")
	}

	second := svc.getSignalRunner()
	if second != first {
		t.Fatal("expected getSignalRunner to return singleton instance")
	}
}

func TestService_GetSignalRunner_UsesInjectedRunner(t *testing.T) {
	fake := &fakeSignalRunner{}
	svc := &Service{signalRunner: fake}

	if got := svc.getSignalRunner(); got != fake {
		t.Fatal("expected injected signal runner to be returned")
	}
}
