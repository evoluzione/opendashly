package retention

import "testing"

func TestPartitionSelection(t *testing.T) {
	parts := []partition{
		{Table: "otel_logs", ID: "20260901", Rows: 10, Bytes: 100},
		{Table: "otel_traces", ID: "20260901", Rows: 5, Bytes: 50},
		{Table: "otel_traces_trace_id_ts", ID: "20260901", Rows: 3, Bytes: 5},
		{Table: "otel_logs", ID: "20260902", Rows: 10, Bytes: 100},
		{Table: "otel_traces", ID: "20260903", Rows: 5, Bytes: 50},
	}

	before := partitionsBefore(parts, "traces", "20260902")
	if len(before) != 2 || before[0].Table != "otel_traces" || before[1].Table != "otel_traces_trace_id_ts" {
		t.Fatalf("partitionsBefore traces = %+v", before)
	}
	if got := partitionsBefore(parts, "logs", "20260901"); len(got) != 0 {
		t.Fatalf("cutoff day must not be dropped: %+v", got)
	}

	free := partitionsToFree(parts, 120)
	if len(free) != 2 || free[0].ID != "20260901" || free[1].Table != "otel_traces" {
		t.Fatalf("partitionsToFree(120) = %+v", free)
	}
	// Newest partition of each table is always kept.
	for _, p := range partitionsToFree(parts, 1<<60) {
		if (p.Table == "otel_logs" && p.ID == "20260902") || (p.Table == "otel_traces" && p.ID == "20260903") {
			t.Fatalf("dropped newest partition %+v", p)
		}
	}
}
