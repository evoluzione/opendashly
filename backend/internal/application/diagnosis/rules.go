package diagnosis

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
)

// ponytail: fixed thresholds, move to workspace settings if teams need to tune them.
const (
	errorRateWarnPct     = 5.0
	errorRateCriticalPct = 10.0
	errorRateMinPct      = 1.0
	errorRateSurgeFactor = 2.0
	minRequests          = 20

	trafficDropFactor  = 0.5
	minBaselineRequest = 10 // per minute

	latencyRegressionFactor = 1.5
	latencyRegressionMinMs  = 100.0
	latencyMinCount         = 10

	hotspotMinErrors   = 5
	surgeFactor        = 3.0
	logPatternMinCount = 3
	maxPatternLen      = 160
)

const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

const (
	KindErrorRate   = "error_rate"
	KindTrafficDrop = "traffic_drop"
	KindNoTraffic   = "no_traffic"
	KindLatency     = "latency"
	KindHotspot     = "hotspot"
	KindLogPattern  = "log_pattern"
	KindRootCause   = "root_cause"
)

// Finding is one anomaly detected by a rule. Current/Baseline carry the
// compared magnitude (percent, ms, count) depending on Kind.
type Finding struct {
	Severity string
	Kind     string
	Service  string
	Endpoint string
	Current  float64
	Baseline float64
	Count    int
	Detail   string
	TraceIDs []string
	// NoBaseline marks findings judged on absolute values because the
	// comparison window had no data.
	NoBaseline bool
}

func dashboardFindings(cur, base *metrics.DashboardResponse, window time.Duration) []Finding {
	out := []Finding{}
	curReq := cur.Satisfaction.Throughput.TotalRequests
	baseReq := base.Satisfaction.Throughput.TotalRequests
	curRate := cur.Satisfaction.ErrorRate
	baseRate := base.Satisfaction.ErrorRate

	// Without baseline data every value would look "new": judge absolute values only.
	noBase := baseReq == 0

	if curReq >= minRequests && noBase {
		if curRate >= errorRateWarnPct {
			sev := SeverityWarning
			if curRate >= errorRateCriticalPct {
				sev = SeverityCritical
			}
			out = append(out, Finding{Severity: sev, Kind: KindErrorRate, Current: curRate, NoBaseline: true})
		}
	} else if curReq >= minRequests {
		surge := curRate >= errorRateMinPct && curRate >= baseRate*errorRateSurgeFactor
		if curRate >= errorRateWarnPct || surge {
			sev := SeverityWarning
			if curRate >= errorRateCriticalPct {
				sev = SeverityCritical
			}
			out = append(out, Finding{Severity: sev, Kind: KindErrorRate, Current: curRate, Baseline: baseRate})
		}
	}

	baseRPM := float64(baseReq) / window.Minutes()
	if baseRPM >= minBaselineRequest {
		if curReq == 0 {
			out = append(out, Finding{Severity: SeverityCritical, Kind: KindNoTraffic, Current: 0, Baseline: float64(baseReq)})
		} else if float64(curReq) < float64(baseReq)*trafficDropFactor {
			out = append(out, Finding{Severity: SeverityWarning, Kind: KindTrafficDrop, Current: float64(curReq), Baseline: float64(baseReq)})
		}
	}

	baseLatency := map[string]float64{}
	for _, e := range base.Hotspots.SlowestEndpoints {
		baseLatency[e.Service+"|"+e.Endpoint] = e.P95
	}
	for _, e := range cur.Hotspots.SlowestEndpoints {
		prev, ok := baseLatency[e.Service+"|"+e.Endpoint]
		if !ok || e.Count < latencyMinCount {
			continue
		}
		if e.P95 >= prev*latencyRegressionFactor && e.P95-prev >= latencyRegressionMinMs {
			out = append(out, Finding{Severity: SeverityWarning, Kind: KindLatency, Service: e.Service, Endpoint: e.Endpoint, Current: e.P95, Baseline: prev, Count: int(e.Count)})
		}
	}

	baseErrors := map[string]int64{}
	for _, h := range base.Hotspots.ErrorHotspots {
		baseErrors[h.Service+"|"+h.Endpoint] = h.ErrorCount
	}
	for _, h := range cur.Hotspots.ErrorHotspots {
		if h.ErrorCount < hotspotMinErrors {
			continue
		}
		if noBase {
			if h.ErrorRate >= errorRateWarnPct {
				out = append(out, Finding{Severity: SeverityWarning, Kind: KindHotspot, Service: h.Service, Endpoint: h.Endpoint, Current: float64(h.ErrorCount), Count: int(h.TotalCount), NoBaseline: true})
			}
			continue
		}
		prev := baseErrors[h.Service+"|"+h.Endpoint]
		if prev == 0 || float64(h.ErrorCount) >= float64(prev)*surgeFactor {
			out = append(out, Finding{Severity: SeverityWarning, Kind: KindHotspot, Service: h.Service, Endpoint: h.Endpoint, Current: float64(h.ErrorCount), Baseline: float64(prev), Count: int(h.TotalCount)})
		}
	}
	return out
}

var (
	uuidRe   = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	hexRe    = regexp.MustCompile(`(?i)\b(0x)?[0-9a-f]{8,}\b`)
	numberRe = regexp.MustCompile(`\d+(\.\d+)?`)
	spaceRe  = regexp.MustCompile(`\s+`)
)

// normalizePattern turns a log message into a template so that messages that
// differ only by ids, numbers or hashes are grouped together.
func normalizePattern(body string) string {
	s := uuidRe.ReplaceAllString(body, "*")
	s = hexRe.ReplaceAllString(s, "*")
	s = numberRe.ReplaceAllString(s, "*")
	s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
	s = strings.ReplaceAll(s, "`", "'")
	if len(s) > maxPatternLen {
		s = s[:maxPatternLen] + "…"
	}
	return s
}

type logPattern struct {
	pattern  string
	service  string
	count    int
	traceIDs []string
}

func groupLogs(logs []query.LogEntry) map[string]*logPattern {
	out := map[string]*logPattern{}
	for _, l := range logs {
		p := normalizePattern(l.Body)
		if p == "" {
			continue
		}
		g, ok := out[p]
		if !ok {
			g = &logPattern{pattern: p, service: l.ResourceAttributes["service.name"]}
			out[p] = g
		}
		g.count++
		if l.TraceID != "" && len(g.traceIDs) < 2 && !slices.Contains(g.traceIDs, l.TraceID) {
			g.traceIDs = append(g.traceIDs, l.TraceID)
		}
	}
	return out
}

// logFindings flags error-log patterns that are new or surging vs the baseline.
// ponytail: both sides are samples (most recent N logs), so "new" means "not in
// the baseline sample"; switch to a GROUP BY on otel_logs if that proves noisy.
func logFindings(cur, base []query.LogEntry, noBase bool) []Finding {
	curGroups := groupLogs(cur)
	baseGroups := groupLogs(base)
	out := []Finding{}
	for _, g := range curGroups {
		if g.count < logPatternMinCount {
			continue
		}
		prev := 0
		if b, ok := baseGroups[g.pattern]; ok {
			prev = b.count
		}
		switch {
		case noBase:
			out = append(out, Finding{Severity: SeverityInfo, Kind: KindLogPattern, Service: g.service, Detail: g.pattern, Current: float64(g.count), TraceIDs: g.traceIDs, NoBaseline: true})
		case prev == 0:
			out = append(out, Finding{Severity: SeverityWarning, Kind: KindLogPattern, Service: g.service, Detail: g.pattern, Current: float64(g.count), TraceIDs: g.traceIDs})
		case float64(g.count) >= float64(prev)*surgeFactor:
			out = append(out, Finding{Severity: SeverityInfo, Kind: KindLogPattern, Service: g.service, Detail: g.pattern, Current: float64(g.count), Baseline: float64(prev), TraceIDs: g.traceIDs})
		}
	}
	return out
}

func isErrorStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ERROR", "STATUS_CODE_ERROR", "2":
		return true
	}
	return false
}

// rootCauseSpan returns the deepest errored span: an error span none of whose
// children errored. That is where the failure originated, not where it surfaced.
func rootCauseSpan(spans []query.TraceSpanEntry) (query.TraceSpanEntry, bool) {
	errorParents := map[string]bool{}
	for _, s := range spans {
		if isErrorStatus(s.Status) && s.ParentSpanID != "" {
			errorParents[s.ParentSpanID] = true
		}
	}
	var best query.TraceSpanEntry
	found := false
	for _, s := range spans {
		if !isErrorStatus(s.Status) || errorParents[s.SpanID] {
			continue
		}
		if !found || s.StartTime.Before(best.StartTime) {
			best, found = s, true
		}
	}
	return best, found
}

// spanErrorDetail picks the most useful error description a span carries.
func spanErrorDetail(s query.TraceSpanEntry) string {
	parts := []string{}
	add := func(attrs map[string]string) {
		for _, k := range []string{"exception.type", "exception.message"} {
			if v := strings.TrimSpace(attrs[k]); v != "" {
				parts = append(parts, v)
			}
		}
	}
	add(s.Attributes)
	for _, e := range s.Events {
		if len(parts) > 0 {
			break
		}
		if e.Name == "exception" {
			add(e.Attributes)
		}
	}
	if len(parts) == 0 && s.StatusMessage != "" {
		parts = append(parts, s.StatusMessage)
	}
	for _, k := range []string{"http.response.status_code", "http.status_code"} {
		if v := s.Attributes[k]; v != "" {
			parts = append(parts, "HTTP "+v)
			break
		}
	}
	detail := strings.Join(parts, ": ")
	if len(detail) > maxPatternLen {
		detail = detail[:maxPatternLen] + "…"
	}
	return detail
}

// rootCauseFindings groups the root-cause spans of several error traces.
func rootCauseFindings(traces map[string][]query.TraceSpanEntry) []Finding {
	groups := map[string]*Finding{}
	order := []string{}
	ids := make([]string, 0, len(traces))
	for id := range traces {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		span, ok := rootCauseSpan(traces[id])
		if !ok {
			continue
		}
		detail := spanErrorDetail(span)
		key := span.Service + "|" + span.Name + "|" + normalizePattern(detail)
		f, ok := groups[key]
		if !ok {
			f = &Finding{Severity: SeverityWarning, Kind: KindRootCause, Service: span.Service, Endpoint: span.Name, Detail: detail}
			groups[key] = f
			order = append(order, key)
		}
		f.Count++
		f.TraceIDs = append(f.TraceIDs, id)
		if f.Count > 1 {
			f.Severity = SeverityCritical
		}
	}
	out := make([]Finding, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out
}

// slowestSpans returns the n longest spans of a trace (its likely critical path).
func slowestSpans(spans []query.TraceSpanEntry, n int) []query.TraceSpanEntry {
	sorted := append([]query.TraceSpanEntry(nil), spans...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Duration > sorted[j].Duration })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

var severityRank = map[string]int{SeverityCritical: 0, SeverityWarning: 1, SeverityInfo: 2}

// kindRank puts service-wide signals first, then where errors originate, then
// per-endpoint and per-message details.
var kindRank = map[string]int{KindNoTraffic: 0, KindErrorRate: 1, KindTrafficDrop: 2, KindRootCause: 3, KindHotspot: 4, KindLatency: 5, KindLogPattern: 6}

func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if severityRank[f[i].Severity] != severityRank[f[j].Severity] {
			return severityRank[f[i].Severity] < severityRank[f[j].Severity]
		}
		if kindRank[f[i].Kind] != kindRank[f[j].Kind] {
			return kindRank[f[i].Kind] < kindRank[f[j].Kind]
		}
		return magnitude(f[i]) > magnitude(f[j])
	})
}

func magnitude(f Finding) float64 {
	if f.Baseline > 0 {
		return f.Current / f.Baseline
	}
	return f.Current
}
