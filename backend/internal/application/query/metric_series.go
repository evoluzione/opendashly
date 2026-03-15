package query

import "sort"

func buildMetricSeries(rows []metricRow) []MetricSeries {
	if len(rows) == 0 {
		return []MetricSeries{}
	}
	grouped := map[string]*MetricSeries{}
	for _, row := range rows {
		key := row.Name + "|" + row.Unit
		series, ok := grouped[key]
		if !ok {
			series = &MetricSeries{Name: row.Name, Unit: row.Unit}
			grouped[key] = series
		}
		series.Points = append(series.Points, MetricPoint{
			Timestamp: row.Timestamp,
			Value:     row.Value,
		})
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	seriesList := make([]MetricSeries, 0, len(keys))
	for _, key := range keys {
		series := grouped[key]
		sort.Slice(series.Points, func(i, j int) bool {
			return series.Points[i].Timestamp.Before(series.Points[j].Timestamp)
		})
		seriesList = append(seriesList, *series)
	}
	return seriesList
}
