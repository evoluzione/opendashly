// Package diagnosis finds problems in telemetry without an LLM: it compares
// the requested window with the previous one of equal length using the
// dashboard rollups, samples error logs and error traces, and renders a
// markdown report with the evidence.
package diagnosis

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/infrastructure/querysql"
)

const (
	maxPromptLen     = 4000
	defaultWindow    = time.Hour
	maxWindow        = 7 * 24 * time.Hour
	logSampleLimit   = 200
	traceSampleLimit = 3
	criticalPathLen  = 5
)

type Runner struct {
	Query   *query.Service
	Spans   *query.TraceSpansService
	Related *query.RelatedService
	Metrics *metrics.Service
}

type Report struct {
	Answer      string       `json:"answer"`
	Steps       []string     `json:"steps,omitempty"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
	// Context lets the next message refer to this answer.
	Context *Context `json:"context,omitempty"`
	// Lang is the language of the answer ("it" or "en"), so the UI labels match it.
	Lang string `json:"lang,omitempty"`
	// Links open the data behind the answer in the search page or download it.
	Links []Link `json:"links,omitempty"`
	// Rows are the actions of each numbered item in the answer, in order.
	Rows []Row `json:"rows,omitempty"`
}

// Row lets the UI analyze or open one item of the answer's numbered list.
type Row struct {
	Prompt string `json:"prompt"`
	Link   *Link  `json:"link,omitempty"`
}

// Suggestion is a follow-up the UI offers as a button: Prompt is written so
// that parseScope understands it.
type Suggestion struct {
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
	// Kind places the button: inline ones act on the answer and sit inside
	// it, rerun ones repeat the analysis on another window or service (the UI
	// prefixes them with "Ripeti per:"), the others are plain next questions.
	Kind string `json:"kind,omitempty"`
}

const (
	sugInline = "inline"
	sugRerun  = "rerun"
)

type Scope struct {
	From    time.Time
	To      time.Time
	Service string
	TraceID string
	// Explicit reports whether the prompt named a time window.
	Explicit bool
	// WindowMentioned is true also for unusable windows ("ultimi 0 minuti").
	WindowMentioned bool
	// Mentions counts the services named; Service is set only when it is one.
	Mentions int
	// Today marks a "since midnight" window, Yesterday the previous calendar day.
	Today     bool
	Yesterday bool
}

func (s Scope) window() time.Duration { return s.To.Sub(s.From) }

func (s Scope) baseline() (time.Time, time.Time) { return s.From.Add(-s.window()), s.From }

// Run answers one chat message. prev is the context of the previous answer
// (nil for the first message). Analysis checks are independent: a failing one
// is reported in the answer instead of failing the request.
func (r *Runner) Run(ctx context.Context, prompt, locale string, now time.Time, prev *Context) (*Report, error) {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > maxPromptLen {
		return nil, fmt.Errorf("prompt too long")
	}
	var services []string
	if r.Query != nil {
		services, _ = r.Query.ListServices(ctx)
	}
	prev = prev.sanitize(services)
	// Answer in the language the message is written in; fall back to the UI locale.
	if lang := detectLanguage(normalize(prompt)); lang != "" {
		locale = lang
	}
	t, p := textsFor(locale), phrasingFor(locale)
	rep, err := r.answer(ctx, prompt, services, now, prev, t, p)
	if rep != nil {
		rep.Lang = p.pick("it", "en")
	}
	return rep, err
}

func (r *Runner) answer(ctx context.Context, prompt string, services []string, now time.Time, prev *Context, t texts, p phrasing) (*Report, error) {
	req := understand(prompt, services, now, prev)

	switch req.action {
	case actReply:
		return &Report{Answer: p.reply(req.intent), Suggestions: starterSuggestions(p, req.intent), Context: prev}, nil
	case actUnsupported:
		return &Report{Answer: p.unsupportedText(req.unsupported), Suggestions: starterSuggestions(p, intentUnclear), Context: prev}, nil
	case actNoMatch:
		if req.scope.Service != "" && prev != nil {
			w := prev.scopeAt(now)
			sug := Suggestion{Label: p.pick("Analizza ", "Analyze ") + req.scope.Service, Prompt: p.windowPrompt(req.scope.Service, w.window()), Kind: sugInline}
			return &Report{Answer: p.noItemFor(req.scope.Service), Suggestions: []Suggestion{sug}, Context: prev}, nil
		}
		n := 0
		if prev != nil {
			n = len(prev.Items)
		}
		return &Report{Answer: p.noItem(n), Suggestions: detailSuggestions(p), Context: prev}, nil
	case actAmbiguousRef:
		nums := make([]string, len(req.refCandidates))
		sugs := []Suggestion{}
		for i, idx := range req.refCandidates {
			nums[i] = fmt.Sprint(idx + 1)
			sugs = append(sugs, Suggestion{Label: fmt.Sprintf(p.pick("Apri il %d", "Open #%d"), idx+1), Prompt: fmt.Sprintf(p.pick("apri il %d", "open #%d"), idx+1), Kind: sugInline})
		}
		answer := fmt.Sprintf(p.pick("Più punti corrispondono (%s): quale apro?", "Several items match (%s): which one should I open?"), strings.Join(nums, ", "))
		return &Report{Answer: answer, Suggestions: sugs, Context: prev}, nil
	case actNoContextRef:
		return &Report{Answer: p.noContextText(), Suggestions: starterSuggestions(p, intentUnclear)}, nil
	case actOpenItem:
		idx := resolveRef(req.ref, prev.Items)
		if idx < 0 {
			return &Report{Answer: p.noItem(len(prev.Items)), Suggestions: detailSuggestions(p), Context: prev}, nil
		}
		item := prev.Items[idx]
		var rep *Report
		if item.TraceID != "" {
			rep = r.traceReport(ctx, item.TraceID, now, t, p, req.detail)
			// Keep the analysis window, so "e la latenza?" stays on it.
			w := prev.scopeAt(now)
			rep.Context.From, rep.Context.To, rep.Context.Today, rep.Context.Yesterday = w.From, w.To, w.Today, w.Yesterday
		} else {
			scope := prev.scopeAt(now)
			if item.Service != "" {
				scope.Service = item.Service
			}
			rep = r.windowReport(ctx, scope, t, p, focusOfKind(item.Kind), req.detail, req.infraNote)
		}
		rep.Steps = append([]string{fmt.Sprintf(p.stepOpenItemText(), idx+1)}, rep.Steps...)
		return rep, nil
	case actLinks:
		if req.scope.TraceID == "" {
			if q := p.askMissing(req, services, prev); q != nil {
				return q, nil
			}
		}
		return p.linksReport(req, prev), nil
	case actMeasure:
		if q := p.askMissing(req, services, prev); q != nil {
			return q, nil
		}
		return r.measureReport(ctx, req.scope, req.measure, t, p, req.detail), nil
	case actDetails:
		if prev.Measure != nil {
			return r.measureReport(ctx, prev.scopeAt(now), *prev.Measure, t, p, true), nil
		}
		if prev.TraceID != "" {
			return r.traceReport(ctx, prev.TraceID, now, t, p, true), nil
		}
		return r.windowReport(ctx, prev.scopeAt(now), t, p, prev.Focus, true, req.infraNote), nil
	case actFollowUp:
		return r.windowReport(ctx, req.scope, t, p, req.focus, req.detail, req.infraNote), nil
	}
	if req.scope.TraceID != "" {
		return r.traceReport(ctx, req.scope.TraceID, now, t, p, req.detail), nil
	}
	if q := p.askMissing(req, services, prev); q != nil {
		return q, nil
	}
	return r.windowReport(ctx, req.scope, t, p, req.focus, req.detail, req.infraNote), nil
}

// analysis is everything collected for a window, before rendering.
type analysis struct {
	findings []Finding
	cur      *metrics.DashboardResponse
	base     *metrics.DashboardResponse
	failed   []string
	steps    []string
	// metricsBusy: the rollup queries failed (usually the backend is under
	// memory pressure); the answer says so instead of showing zeros.
	metricsBusy bool
	// errorLogs / errorTraces: how many were found, so links never lead to
	// an empty search page.
	errorLogs, errorTraces int
}

func (r *Runner) analyze(ctx context.Context, scope Scope, t texts) analysis {
	a := analysis{findings: []Finding{}, failed: []string{}}
	baseFrom, baseTo := scope.baseline()
	a.steps = []string{fmt.Sprintf(t.stepScope, formatRange(scope.From, scope.To), serviceLabel(t, scope.Service))}

	if r.Metrics != nil {
		// Route-level rollups (ids in paths collapsed) are lighter than the full
		// dashboard and group the same endpoint called with different ids.
		filter := metrics.RouteFilter{Service: scope.Service}
		curTotal, curRoutes, errCur := r.Metrics.RouteStatsFor(ctx, scope.From, scope.To, filter, routeStatsLimit)
		baseTotal, baseRoutes, errBase := r.Metrics.RouteStatsFor(ctx, baseFrom, baseTo, filter, routeStatsLimit)
		a.steps = append(a.steps, fmt.Sprintf(t.stepMetrics, formatRange(baseFrom, baseTo)))
		if errCur == nil && errBase == nil {
			a.cur, a.base = routeDashboard(curTotal, curRoutes), routeDashboard(baseTotal, baseRoutes)
			found := dashboardFindings(a.cur, a.base, scope.window())
			a.findings = append(a.findings, found...)
			if len(found) > 0 {
				a.steps = append(a.steps, fmt.Sprintf(t.stepMetricsBad, len(found)))
			} else {
				a.steps = append(a.steps, t.stepMetricsOk)
			}
		} else {
			a.metricsBusy = true
		}
	}

	curLogs, errCur := r.errorLogs(ctx, scope.Service, scope.From, scope.To)
	baseLogs, errBase := r.errorLogs(ctx, scope.Service, baseFrom, baseTo)
	if errCur == nil && errBase == nil {
		noBase := a.base != nil && a.base.Satisfaction.Throughput.TotalRequests == 0 && len(baseLogs) == 0
		a.errorLogs = len(curLogs)
		found := logFindings(curLogs, baseLogs, noBase)
		a.findings = append(a.findings, found...)
		a.steps = append(a.steps, fmt.Sprintf(t.stepLogs, len(curLogs), len(found)))
	} else {
		a.failed = append(a.failed, t.checkLogs)
	}

	traces, err := r.errorTraces(ctx, scope)
	if err == nil {
		spans := map[string][]query.TraceSpanEntry{}
		a.errorTraces = len(traces)
		for _, id := range traces {
			if s, err := r.Spans.Spans(ctx, id); err == nil {
				spans[id] = s
			}
		}
		if len(spans) > 0 {
			a.steps = append(a.steps, fmt.Sprintf(t.stepTraces, len(spans)))
		} else {
			a.steps = append(a.steps, t.stepTracesNone)
		}
		a.findings = append(a.findings, rootCauseFindings(spans)...)
	} else {
		a.failed = append(a.failed, t.checkTraces)
	}

	sortFindings(a.findings)
	if len(a.findings) > 0 {
		a.steps = append(a.steps, fmt.Sprintf(t.stepRank, len(a.findings)))
	}
	return a
}

func (r *Runner) windowReport(ctx context.Context, scope Scope, t texts, p phrasing, focus string, detail, infraNote bool) *Report {
	a := r.analyze(ctx, scope, t)
	out := &Context{From: scope.From, To: scope.To, Service: scope.Service, Focus: focus, Today: scope.Today, Yesterday: scope.Yesterday}
	var answer string
	if detail {
		answer = renderWindow(t, p, scope, a.findings, a.cur, a.failed)
		out.Items = contextItems(a.findings, maxFindings)
	} else {
		answer, out.Items = conciseWindow(p, scope, focus, a)
	}
	if infraNote {
		answer = p.infraNoteText() + "\n\n" + answer
	}
	links := p.focusLinks(scope, focus, a.errorLogs > 0, a.errorTraces > 0)
	sugs := windowSuggestions(p, scope, out.Items, detail)
	if a.cur != nil && a.cur.Satisfaction.Throughput.TotalRequests == 0 && len(a.findings) == 0 {
		sugs = []Suggestion{{Label: p.pick("oggi", "today"), Prompt: p.pick("e oggi?", "and today?"), Kind: sugRerun},
			{Label: p.pick("ultime 24 ore", "last 24 hours"), Prompt: p.pick("e nelle ultime 24 ore?", "and in the last 24 hours?"), Kind: sugRerun}}
	}
	return &Report{Answer: answer, Steps: a.steps, Suggestions: sugs, Context: out, Links: links, Rows: p.rows(scope, out.Items)}
}

func (r *Runner) traceReport(ctx context.Context, traceID string, now time.Time, t texts, p phrasing, detail bool) *Report {
	failed := []string{}
	var spans []query.TraceSpanEntry
	if r.Spans != nil {
		var err error
		if spans, err = r.Spans.Spans(ctx, traceID); err != nil {
			failed = append(failed, t.checkTraces)
		}
	}
	var logs []query.LogEntry
	if r.Related != nil {
		if res, err := r.Related.Related(ctx, traceID); err == nil {
			for _, item := range res.Logs {
				if l, ok := item.(query.LogEntry); ok && isErrorSeverity(l.Severity) {
					logs = append(logs, l)
				}
			}
		} else {
			failed = append(failed, t.checkLogs)
		}
	}
	steps := []string{fmt.Sprintf(t.stepTrace, len(spans), len(logs))}
	root, hasRoot := rootCauseSpan(spans)
	if hasRoot {
		steps = append(steps, fmt.Sprintf(t.stepTraceRoot, root.Name, root.Service))
	}
	steps = append(steps, t.stepTraceSlow)
	answer := conciseTrace(p, spans, logs, failed)
	if detail {
		answer = renderTrace(t, p, traceID, spans, logs, failed)
	}
	out := &Context{From: now.Add(-defaultWindow), To: now, TraceID: traceID}
	if hasRoot {
		out.Service = root.Service
	}
	return &Report{Answer: answer, Steps: steps, Suggestions: traceSuggestions(p, root, hasRoot, detail), Context: out,
		Links: []Link{p.traceLink(traceID)}}
}

func (r *Runner) errorLogs(ctx context.Context, service string, from, to time.Time) ([]query.LogEntry, error) {
	if r.Query == nil {
		return nil, fmt.Errorf("query service unavailable")
	}
	filters := []querysql.FilterItem{{Key: "severity", Operator: "=", Value: "ERROR,FATAL"}}
	if service != "" {
		filters = append(filters, querysql.FilterItem{Connector: "AND", Key: "service.name", Operator: "=", Value: service})
	}
	res, err := r.Query.Run(ctx, query.QueryRequest{
		Signals:    []string{"logs"},
		TimeRange:  query.TimeRange{From: from, To: to},
		FilterList: filters,
		Limit:      logSampleLimit,
	})
	if err != nil {
		return nil, err
	}
	if msg, ok := res.SignalErrors["logs"]; ok {
		return nil, fmt.Errorf("%s", msg)
	}
	out := make([]query.LogEntry, 0, len(res.Results.Logs))
	for _, item := range res.Results.Logs {
		if l, ok := item.(query.LogEntry); ok {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *Runner) errorTraces(ctx context.Context, scope Scope) ([]string, error) {
	if r.Spans == nil {
		return nil, fmt.Errorf("trace service unavailable")
	}
	return r.Spans.RecentErrorTraceIDs(ctx, scope.From, scope.To, scope.Service, traceSampleLimit)
}

func isErrorSeverity(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ERROR", "ERR", "FATAL", "CRITICAL", "SEVERE", "ALERT", "EMERGENCY":
		return true
	}
	return false
}

// routeDashboard shapes route stats like the dashboard response the rules read:
// totals, the slowest routes by p95 and the routes with errors.
func routeDashboard(total metrics.RouteStats, routes []metrics.RouteStats) *metrics.DashboardResponse {
	d := &metrics.DashboardResponse{Health: metrics.DashboardHealth{Status: "ok"}}
	d.Satisfaction.Throughput.TotalRequests = total.Count
	d.Satisfaction.Throughput.TotalErrors = total.Errors
	if total.Count > 0 {
		d.Satisfaction.ErrorRate = float64(total.Errors) / float64(total.Count) * 100
	}
	slow := append([]metrics.RouteStats(nil), routes...)
	sort.SliceStable(slow, func(i, j int) bool { return slow[i].P95 > slow[j].P95 })
	for i, r := range slow {
		if i == 10 {
			break
		}
		d.Hotspots.SlowestEndpoints = append(d.Hotspots.SlowestEndpoints, metrics.EndpointLatency{
			Endpoint: r.Route, Service: r.Service, AvgMs: r.AvgMs, P50: r.P50, P95: r.P95, P99: r.P99, Count: r.Count})
	}
	hot := []metrics.RouteStats{}
	for _, r := range routes {
		if r.Errors > 0 {
			hot = append(hot, r)
		}
	}
	sort.SliceStable(hot, func(i, j int) bool { return hot[i].Errors > hot[j].Errors })
	for i, r := range hot {
		if i == 10 {
			break
		}
		d.Hotspots.ErrorHotspots = append(d.Hotspots.ErrorHotspots, metrics.ErrorHotspot{
			Endpoint: r.Route, Service: r.Service, ErrorCount: r.Errors, TotalCount: r.Count,
			ErrorRate: float64(r.Errors) / float64(max(r.Count, 1)) * 100})
	}
	return d
}
