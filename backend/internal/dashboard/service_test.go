package dashboard

import (
	"context"
	"testing"
)

type fakeRepo struct {
	getResult []ChartSetting
	upserted  []ChartSetting
}

func (f *fakeRepo) GetSettings(context.Context, string) ([]ChartSetting, error) {
	out := make([]ChartSetting, len(f.getResult))
	copy(out, f.getResult)
	return out, nil
}

func (f *fakeRepo) UpsertSettings(_ context.Context, _ string, settings []ChartSetting, _ string) error {
	f.upserted = make([]ChartSetting, len(settings))
	copy(f.upserted, settings)
	return nil
}

func TestGetSettingsReturnsDefaultLayoutWhenRepoEmpty(t *testing.T) {
	repo := &fakeRepo{}
	svc := &Service{Repo: repo}

	got, err := svc.GetSettings(context.Background(), "default")
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if len(got) != len(defaultChartOrder) {
		t.Fatalf("GetSettings() length = %d, want %d", len(got), len(defaultChartOrder))
	}
	for i, def := range defaultChartOrder {
		if got[i] != def {
			t.Fatalf("GetSettings()[%d] = %+v, want %+v", i, got[i], def)
		}
	}
}

func TestGetSettingsNormalizesLegacyStoredValues(t *testing.T) {
	repo := &fakeRepo{
		getResult: []ChartSetting{
			{Key: "apdex_gauge", Enabled: false, Order: 0, X: 0, Y: 0, W: 0, H: 0},
		},
	}
	svc := &Service{Repo: repo}

	got, err := svc.GetSettings(context.Background(), "default")
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if len(got) == 0 {
		t.Fatal("GetSettings() returned no settings")
	}
	first := got[0]
	if first.Key != "apdex_gauge" {
		t.Fatalf("first key = %s, want apdex_gauge", first.Key)
	}
	if first.Enabled {
		t.Fatalf("first enabled = %v, want false", first.Enabled)
	}
	if first.Order != 10 || first.W != 2 || first.H != 2 {
		t.Fatalf("first normalized = %+v, want order=10 w=2 h=2", first)
	}
}

func TestUpdateSettingsFiltersUnknownAndNormalizesLayout(t *testing.T) {
	repo := &fakeRepo{}
	svc := &Service{Repo: repo}

	_, err := svc.UpdateSettings(context.Background(), "default", "admin", []ChartSetting{
		{Key: "unknown_chart", Enabled: true, Order: 10, X: 0, Y: 0, W: 4, H: 2},
		{Key: "apdex_gauge", Enabled: true, Order: 0, X: -4, Y: -8, W: 99, H: 0},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("UpsertSettings() got %d settings, want 1", len(repo.upserted))
	}
	got := repo.upserted[0]
	if got.Key != "apdex_gauge" {
		t.Fatalf("upsert key = %s, want apdex_gauge", got.Key)
	}
	if got.Order != 10 {
		t.Fatalf("order = %d, want 10", got.Order)
	}
	if got.X != 0 || got.Y != 0 {
		t.Fatalf("position = (%d,%d), want (0,0)", got.X, got.Y)
	}
	if got.W != 6 || got.H != 2 {
		t.Fatalf("size = (%d,%d), want (6,2)", got.W, got.H)
	}
}
