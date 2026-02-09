package dashboard

type ChartSetting struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Order   int32  `json:"order"`
}

type SettingsResponse struct {
	Settings []ChartSetting `json:"settings"`
}

type UpdateSettingsRequest struct {
	Settings []ChartSetting `json:"settings"`
}
