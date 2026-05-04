package builders

import "testing"

func TestFilterListForSignal_TracesSkipsLogOnlyKeys(t *testing.T) {
	in := []FilterItem{
		{Key: "body", Operator: "contains", Value: "boom"},
		{Key: "severity", Operator: "=", Value: "ERROR"},
		{Key: "service.name", Operator: "=", Value: "checkout"},
	}

	out := filterListForSignal("traces", in)
	if len(out) != 1 || out[0].Key != "service.name" {
		t.Fatalf("unexpected filtered output: %#v", out)
	}
}

