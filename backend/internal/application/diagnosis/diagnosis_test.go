package diagnosis

import (
	"context"
	"strings"
	"testing"
	"time"

	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
)

var testNow = time.Date(2026, 9, 24, 15, 0, 0, 0, time.UTC)

func TestParseScope(t *testing.T) {
	services := []string{"api", "api-gateway", "checkout"}
	cases := []struct {
		prompt  string
		window  time.Duration
		service string
		traceID string
	}{
		{"", time.Hour, "", ""},
		{"errori negli ultimi 15 minuti", 15 * time.Minute, "", ""},
		{"what happened in the last 2h on checkout", 2 * time.Hour, "checkout", ""},
		{"api-gateway ultime 24 ore", 24 * time.Hour, "api-gateway", ""},
		{"errori oggi", 15 * time.Hour, "", ""},
		{"ultimi 30 giorni", maxWindow, "", ""},
		{"trace 4bf92f3577b34da6a3ce929d0e0e4736 perché fallisce?", time.Hour, "", "4bf92f3577b34da6a3ce929d0e0e4736"},
		{"http 500 su api", time.Hour, "api", ""},
	}
	for _, c := range cases {
		s := parseScope(c.prompt, services, testNow)
		if s.window() != c.window || s.Service != c.service || s.TraceID != c.traceID || !s.To.Equal(testNow) {
			t.Errorf("parseScope(%q) = window %v service %q trace %q, want %v %q %q", c.prompt, s.window(), s.Service, s.TraceID, c.window, c.service, c.traceID)
		}
	}
}

func dash(requests int64, errorRate float64, slow []metrics.EndpointLatency, hot []metrics.ErrorHotspot) *metrics.DashboardResponse {
	d := &metrics.DashboardResponse{}
	d.Satisfaction.Throughput.TotalRequests = requests
	d.Satisfaction.ErrorRate = errorRate
	d.Hotspots.SlowestEndpoints = slow
	d.Hotspots.ErrorHotspots = hot
	return d
}

func kinds(f []Finding) string {
	out := []string{}
	for _, x := range f {
		out = append(out, x.Severity+":"+x.Kind)
	}
	return strings.Join(out, ",")
}

func TestDashboardFindings(t *testing.T) {
	slowBase := []metrics.EndpointLatency{{Service: "api", Endpoint: "GET /a", P95: 100, Count: 50}}
	slowCur := []metrics.EndpointLatency{{Service: "api", Endpoint: "GET /a", P95: 400, Count: 50}}
	hot := []metrics.ErrorHotspot{{Service: "api", Endpoint: "POST /pay", ErrorCount: 12, TotalCount: 40, ErrorRate: 30}}
	cases := []struct {
		name      string
		cur, base *metrics.DashboardResponse
		want      string
	}{
		{"healthy", dash(1000, 0.5, slowBase, nil), dash(1000, 0.4, slowBase, nil), ""},
		{"error surge", dash(1000, 3, nil, nil), dash(1000, 1, nil, nil), "warning:error_rate"},
		{"no baseline, low error rate", dash(1000, 1.3, nil, nil), dash(0, 0, nil, nil), ""},
		{"no baseline, high error rate", dash(1000, 7, nil, nil), dash(0, 0, nil, nil), "warning:error_rate"},
		{"no baseline, hotspot", dash(1000, 0, nil, hot), dash(0, 0, nil, nil), "warning:hotspot"},
		{"no baseline, low-rate hotspot", dash(1000, 0, nil, []metrics.ErrorHotspot{{Service: "api", Endpoint: "GET /b", ErrorCount: 6, TotalCount: 1000, ErrorRate: 0.6}}), dash(0, 0, nil, nil), ""},
		{"critical error rate", dash(1000, 12, nil, nil), dash(1000, 11, nil, nil), "critical:error_rate"},
		{"too few requests", dash(5, 50, nil, nil), dash(5, 0, nil, nil), ""},
		{"no traffic", dash(0, 0, nil, nil), dash(1200, 0, nil, nil), "critical:no_traffic"},
		{"traffic drop", dash(400, 0, nil, nil), dash(1200, 0, nil, nil), "warning:traffic_drop"},
		{"latency regression", dash(1000, 0, slowCur, nil), dash(1000, 0, slowBase, nil), "warning:latency"},
		{"new hotspot", dash(1000, 0, nil, hot), dash(1000, 0, nil, nil), "warning:hotspot"},
		{"steady hotspot", dash(1000, 0, nil, hot), dash(1000, 0, nil, hot), ""},
	}
	for _, c := range cases {
		if got := kinds(dashboardFindings(c.cur, c.base, time.Hour)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestNormalizePattern(t *testing.T) {
	a := normalizePattern("order 1234 failed for user 550e8400-e29b-41d4-a716-446655440000 after 3.5s")
	b := normalizePattern("order 99 failed for user 123e4567-e89b-12d3-a456-426614174000 after 12s")
	if a != b || a != "order * failed for user * after *s" {
		t.Fatalf("patterns differ: %q vs %q", a, b)
	}
}

func logs(body string, n int) []query.LogEntry {
	out := make([]query.LogEntry, n)
	for i := range out {
		out[i] = query.LogEntry{Body: body, TraceID: "t1", ResourceAttributes: map[string]string{"service.name": "api"}}
	}
	return out
}

func TestLogFindings(t *testing.T) {
	cur := append(logs("db timeout after 30s", 5), logs("rare thing", 1)...)
	if got := kinds(logFindings(cur, nil, false)); got != "warning:log_pattern" {
		t.Fatalf("new pattern: got %q", got)
	}
	if got := kinds(logFindings(cur, logs("db timeout after 10s", 1), false)); got != "info:log_pattern" {
		t.Fatalf("surging pattern: got %q", got)
	}
	if got := kinds(logFindings(cur, logs("db timeout after 10s", 4), false)); got != "" {
		t.Fatalf("steady pattern: got %q", got)
	}
}

func TestRootCause(t *testing.T) {
	t0 := testNow
	spans := []query.TraceSpanEntry{
		{SpanID: "a", Name: "GET /checkout", Service: "gateway", Status: "Error", StartTime: t0, Duration: 9e8},
		{SpanID: "b", ParentSpanID: "a", Name: "charge", Service: "payments", Status: "STATUS_CODE_ERROR", StartTime: t0.Add(time.Millisecond), Duration: 8e8,
			Events: []query.TraceSpanEvent{{Name: "exception", Attributes: map[string]string{"exception.type": "TimeoutError"}}}},
		{SpanID: "c", ParentSpanID: "a", Name: "render", Service: "gateway", Status: "Ok", StartTime: t0, Duration: 1e6},
	}
	root, ok := rootCauseSpan(spans)
	if !ok || root.SpanID != "b" || spanErrorDetail(root) != "TimeoutError" {
		t.Fatalf("root cause = %+v ok=%v", root, ok)
	}
	f := rootCauseFindings(map[string][]query.TraceSpanEntry{"t1": spans, "t2": spans})
	if len(f) != 1 || f[0].Count != 2 || f[0].Severity != SeverityCritical {
		t.Fatalf("grouped findings = %+v", f)
	}
	if got := slowestSpans(spans, 2); got[0].SpanID != "a" || got[1].SpanID != "b" {
		t.Fatalf("slowest = %v", got)
	}
}

func TestRender(t *testing.T) {
	scope := parseScope("", nil, testNow)
	findings := []Finding{
		{Severity: SeverityInfo, Kind: KindLogPattern, Detail: "x", Current: 9},
		{Severity: SeverityCritical, Kind: KindRootCause, Service: "payments", Endpoint: "charge", Count: 2, Detail: "TimeoutError", TraceIDs: []string{"t1", "t2"}},
	}
	sortFindings(findings)
	out := renderWindow(textsFor("it"), scope, findings, dash(10, 1, nil, nil), []string{"log di errore"})
	for _, want := range []string{"Problemi trovati (2)", "`t1`", "Cosa controllare", "Controlli non completati", "1. 🔴"} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "TimeoutError") > strings.Index(out, "`x`") {
		t.Errorf("critical finding should come first:\n%s", out)
	}
	if !strings.Contains(renderWindow(textsFor("en-US"), scope, nil, nil, nil), "No anomalies") {
		t.Error("english empty report")
	}
}

func TestSuggestionsRoundTrip(t *testing.T) {
	services := []string{"payment-service"}
	scope := parseScope("", services, testNow)
	items := []ContextItem{{Kind: KindRootCause, Service: "payment-service", TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"}}
	prev := &Context{From: scope.From, To: scope.To, Items: items}
	for _, locale := range []string{"it", "en"} {
		sugs := windowSuggestions(phrasingFor(locale), scope, items, false)
		if len(sugs) != 4 {
			t.Fatalf("%s: got %d suggestions: %+v", locale, len(sugs), sugs)
		}
		// Every suggested prompt must be understood, given the answer it follows.
		if r := understand(sugs[0].Prompt, services, testNow, prev); r.action != actDetails {
			t.Errorf("%s details prompt %q -> action %d", locale, sugs[0].Prompt, r.action)
		}
		if r := understand(sugs[1].Prompt, services, testNow, prev); r.action != actOpenItem || resolveRef(r.ref, items) != 0 {
			t.Errorf("%s open prompt %q -> action %d ref %d", locale, sugs[1].Prompt, r.action, r.ref)
		}
		if s := parseScope(sugs[2].Prompt, services, testNow); s.Service != "payment-service" || s.window() != time.Hour {
			t.Errorf("%s service prompt %q parsed as %+v", locale, sugs[2].Prompt, s)
		}
		if s := parseScope(sugs[3].Prompt, services, testNow); s.window() != 24*time.Hour {
			t.Errorf("%s widen prompt %q parsed as %v", locale, sugs[3].Prompt, s.window())
		}
	}
}

func TestClassify(t *testing.T) {
	services := []string{"payment-service", "checkout"}
	cases := map[string]intent{
		"cosa puoi fare?":                    intentHelp,
		"What can you do":                    intentHelp,
		"come funziona?":                     intentHelp,
		"ciao":                               intentGreeting,
		"Buongiorno!":                        intentGreeting,
		"grazie mille":                       intentThanks,
		"mi piace la pizza":                  intentUnclear,
		"errori di oggi":                     intentDiagnose,
		"perché checkout è lento?":           intentDiagnose,
		"payment-service":                    intentDiagnose,
		"ultime 24 ore":                      intentDiagnose,
		"ultima ora":                         intentDiagnose,
		"4bf92f3577b34da6a3ce929d0e0e4736":   intentDiagnose,
		"ci sono problemi?":                  intentDiagnose,
		"why are we getting 502s":            intentDiagnose,
		"why are we getting 502 on checkout": intentDiagnose,
	}
	for prompt, want := range cases {
		if got := classify(prompt, parseScope(prompt, services, testNow)); got != want {
			t.Errorf("classify(%q) = %d, want %d", prompt, got, want)
		}
	}
}

// TestRunRouting drives Run without storage: conversational replies must not
// touch telemetry, and references must use the previous answer.
func TestRunRouting(t *testing.T) {
	r := &Runner{}
	ctx := context.Background()
	help, err := r.Run(ctx, "cosa sai fare?", "it", testNow, nil)
	if err != nil || help.Answer == "" || len(help.Steps) != 0 || len(help.Suggestions) != 3 {
		t.Fatalf("help reply = %+v, %v", help, err)
	}
	en, _ := r.Run(ctx, "what can you do?", "it", testNow, nil)
	if !strings.Contains(en.Answer, "I find") {
		t.Errorf("an English question must get an English answer: %q", en.Answer)
	}
	noCtx, _ := r.Run(ctx, "analizza il primo errore", "it", testNow, nil)
	if !strings.Contains(noCtx.Answer, "Non so a cosa ti riferisci") {
		t.Errorf("reference without context: %q", noCtx.Answer)
	}
	prev := &Context{From: testNow.Add(-time.Hour), To: testNow, Items: []ContextItem{{Kind: KindRootCause, TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"}}}
	open, _ := r.Run(ctx, "analizza il primo errore", "it", testNow, prev)
	if open.Context == nil || open.Context.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("open item must analyze its trace: %+v", open.Context)
	}
	missing, _ := r.Run(ctx, "apri il 5", "it", testNow, prev)
	if !strings.Contains(missing.Answer, "solo 1 punti") {
		t.Errorf("out of range item: %q", missing.Answer)
	}
	chart, _ := r.Run(ctx, "fammi un grafico della latenza", "it", testNow, nil)
	if !strings.Contains(chart.Answer, "grafici") {
		t.Errorf("unsupported chart: %q", chart.Answer)
	}
}
