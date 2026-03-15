package query

// BuildRelatedFilters creates filters for related telemetry.
func BuildRelatedFilters(traceID string) map[string]string {
	return map[string]string{
		"trace_id": traceID,
	}
}
