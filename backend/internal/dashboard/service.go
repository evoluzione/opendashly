package dashboard

import "context"

type Service struct {
	Repo Repository
}

var defaultChartOrder = []ChartSetting{
	{Key: "apdex_gauge", Enabled: true, Order: 10},
	{Key: "error_rate_gauge", Enabled: true, Order: 20},
	{Key: "throughput_gauge", Enabled: true, Order: 30},
	{Key: "latency_distribution", Enabled: true, Order: 40},
	{Key: "latency_percentiles", Enabled: true, Order: 50},
	{Key: "throughput_timeseries", Enabled: true, Order: 60},
	{Key: "error_rate_timeseries", Enabled: true, Order: 70},
	{Key: "status_codes", Enabled: true, Order: 80},
	{Key: "slowest_endpoints", Enabled: true, Order: 90},
	{Key: "top_endpoints_throughput", Enabled: true, Order: 100},
	{Key: "error_hotspots", Enabled: true, Order: 110},
	{Key: "log_volume", Enabled: true, Order: 120},
	{Key: "log_levels", Enabled: true, Order: 130},
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
			if value.Order == 0 {
				value.Order = def.Order
			}
			settings = append(settings, value)
			continue
		}
		settings = append(settings, def)
	}
	return settings, nil
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID string, updatedBy string, settings []ChartSetting) ([]ChartSetting, error) {
	allowed := map[string]struct{}{}
	defaultOrder := map[string]int32{}
	for _, def := range defaultChartOrder {
		allowed[def.Key] = struct{}{}
		defaultOrder[def.Key] = def.Order
	}
	filtered := make([]ChartSetting, 0, len(settings))
	for _, setting := range settings {
		if _, ok := allowed[setting.Key]; !ok {
			continue
		}
		if setting.Order == 0 {
			setting.Order = defaultOrder[setting.Key]
		}
		filtered = append(filtered, setting)
	}
	if err := s.Repo.UpsertSettings(ctx, tenantID, filtered, updatedBy); err != nil {
		return nil, err
	}
	return s.GetSettings(ctx, tenantID)
}
