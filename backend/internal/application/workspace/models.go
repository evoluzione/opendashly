package workspace

const DefaultTitle = "OpenDashly"

type Settings struct {
	Title string `json:"title"`
}

type UpdateSettingsRequest struct {
	Title string `json:"title"`
}
