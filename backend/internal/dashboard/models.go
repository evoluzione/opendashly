package dashboard

type ChartSetting struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Order   int32  `json:"order"`
	X       int32  `json:"x"`
	Y       int32  `json:"y"`
	W       int32  `json:"w"`
	H       int32  `json:"h"`
}

type SettingsResponse struct {
	Settings []ChartSetting `json:"settings"`
}

type UpdateSettingsRequest struct {
	Settings []ChartSetting `json:"settings"`
}
