package query

import (
	"testing"
	"time"
)

func TestBuildMetricSeries_Empty(t *testing.T) {
	got := buildMetricSeries(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty series, got len=%d", len(got))
	}
}

func TestBuildMetricSeries_GroupsAndSorts(t *testing.T) {
	t1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Minute)
	t3 := t1.Add(1 * time.Minute)

	rows := []metricRow{
		{Name: "cpu.usage", Unit: "%", Timestamp: t2, Value: 30},
		{Name: "mem.usage", Unit: "MB", Timestamp: t1, Value: 512},
		{Name: "cpu.usage", Unit: "%", Timestamp: t1, Value: 20},
		{Name: "cpu.usage", Unit: "%", Timestamp: t3, Value: 25},
	}

	got := buildMetricSeries(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 grouped series, got %d", len(got))
	}

	if got[0].Name != "cpu.usage" || got[0].Unit != "%" {
		t.Fatalf("expected cpu series first, got %#v", got[0])
	}
	if len(got[0].Points) != 3 {
		t.Fatalf("expected 3 cpu points, got %d", len(got[0].Points))
	}
	if !got[0].Points[0].Timestamp.Equal(t1) || !got[0].Points[1].Timestamp.Equal(t3) || !got[0].Points[2].Timestamp.Equal(t2) {
		t.Fatalf("expected cpu points sorted by timestamp, got %#v", got[0].Points)
	}

	if got[1].Name != "mem.usage" || got[1].Unit != "MB" {
		t.Fatalf("expected mem series second, got %#v", got[1])
	}
}
