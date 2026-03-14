package dashboard

import "context"

type Service struct {
	Repo Repository
}

const (
	gridColumns = int32(6)
	minW        = int32(1)
	maxW        = int32(6)
	fixedH      = int32(2)
)

var defaultChartOrder = []ChartSetting{
	{Key: "apdex_gauge", Enabled: true, Order: 10, X: 0, Y: 0, W: 2, H: 2},
	{Key: "error_rate_gauge", Enabled: true, Order: 20, X: 2, Y: 0, W: 2, H: 2},
	{Key: "throughput_gauge", Enabled: true, Order: 30, X: 4, Y: 0, W: 2, H: 2},
	{Key: "latency_distribution", Enabled: true, Order: 40, X: 0, Y: 2, W: 3, H: 2},
	{Key: "latency_percentiles", Enabled: true, Order: 50, X: 3, Y: 2, W: 3, H: 2},
	{Key: "throughput_timeseries", Enabled: true, Order: 60, X: 0, Y: 4, W: 3, H: 2},
	{Key: "slowest_endpoints", Enabled: true, Order: 70, X: 3, Y: 4, W: 3, H: 2},
	{Key: "top_endpoints_throughput", Enabled: true, Order: 80, X: 0, Y: 6, W: 3, H: 2},
	{Key: "error_hotspots", Enabled: true, Order: 90, X: 3, Y: 6, W: 3, H: 2},
	{Key: "availability_trend", Enabled: false, Order: 100, X: 0, Y: 8, W: 3, H: 2},
	{Key: "error_budget_burn", Enabled: false, Order: 110, X: 3, Y: 8, W: 3, H: 2},
	{Key: "error_rate_timeseries", Enabled: false, Order: 120, X: 0, Y: 10, W: 3, H: 2},
	{Key: "service_latency_rank", Enabled: false, Order: 130, X: 3, Y: 10, W: 3, H: 2},
	{Key: "slo_compliance", Enabled: false, Order: 140, X: 0, Y: 12, W: 3, H: 2},
	{Key: "service_throughput", Enabled: false, Order: 150, X: 3, Y: 12, W: 3, H: 2},
}

func normalizeLayout(setting ChartSetting, fallback ChartSetting) ChartSetting {
	setting.Key = fallback.Key
	if setting.Order <= 0 {
		setting.Order = fallback.Order
	}

	if setting.X < 0 {
		setting.X = fallback.X
	}
	if setting.Y < 0 {
		setting.Y = fallback.Y
	}
	if setting.W <= 0 {
		setting.W = fallback.W
	}
	if setting.H <= 0 {
		setting.H = fallback.H
	}

	if setting.W < minW {
		setting.W = minW
	}
	if setting.W > maxW {
		setting.W = maxW
	}
	setting.H = fixedH
	if setting.X < 0 {
		setting.X = 0
	}
	if setting.X >= gridColumns {
		setting.X = fallback.X
	}
	if setting.X+setting.W > gridColumns {
		setting.W = gridColumns - setting.X
		if setting.W < minW {
			setting.X = 0
			if gridColumns < minW {
				setting.W = gridColumns
			} else {
				setting.W = minW
			}
		}
	}
	if setting.Y < 0 {
		setting.Y = fallback.Y
	}

	return setting
}

func (s *Service) GetSettings(ctx context.Context, tenantID string) ([]ChartSetting, error) {
	stored, err := s.Repo.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	storedMap := map[string]ChartSetting{}
	for _, setting := range stored {
		storedMap[setting.Key] = setting
	}

	settings := make([]ChartSetting, 0, len(defaultChartOrder))
	for _, def := range defaultChartOrder {
		if value, ok := storedMap[def.Key]; ok {
			settings = append(settings, normalizeLayout(value, def))
			continue
		}
		settings = append(settings, def)
	}
	return settings, nil
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID string, updatedBy string, settings []ChartSetting) ([]ChartSetting, error) {
	allowed := map[string]ChartSetting{}
	for _, def := range defaultChartOrder {
		allowed[def.Key] = def
	}
	inputByKey := map[string]ChartSetting{}
	for _, setting := range settings {
		if _, ok := allowed[setting.Key]; !ok {
			continue
		}
		inputByKey[setting.Key] = setting
	}

	filtered := make([]ChartSetting, 0, len(inputByKey))
	for _, def := range defaultChartOrder {
		incoming, ok := inputByKey[def.Key]
		if !ok {
			continue
		}
		filtered = append(filtered, normalizeLayout(incoming, def))
	}
	if err := s.Repo.UpsertSettings(ctx, tenantID, filtered, updatedBy); err != nil {
		return nil, err
	}
	return s.GetSettings(ctx, tenantID)
}
