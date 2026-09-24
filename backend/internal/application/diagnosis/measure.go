package diagnosis

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"opendashly/backend/internal/application/metrics"
)

// Measure is a precise question ("latenza media delle GET del catalogo",
// "how many errors did payment have today"), answered with a number.
type Measure struct {
	Metric string `json:"metric"`           // latency | errors | requests
	Stat   string `json:"stat,omitempty"`   // avg | p50 | p95 | p99 | max
	Method string `json:"method,omitempty"` // GET, POST, ...
	Path   string `json:"path,omitempty"`   // "/checkout"
}

const (
	metricLatency  = "latency"
	metricErrors   = "errors"
	metricRequests = "requests"

	statAvg = "avg"
	statP50 = "p50"
	statP95 = "p95"
	statP99 = "p99"
	statMax = "max"

	endpointStatsLimit = 500
	measureDetailRows  = 15
)

var (
	measureLatencyStems = []string{"latenz", "latenc", "lent", "slow", "veloc", "speed", "fast", "rapid",
		"quick", "durat", "duration", "prestazion", "performance", "millisecond"}
	measureRequestWords = map[string]bool{"richieste": true, "richiesta": true, "requests": true, "request": true,
		"chiamate": true, "chiamata": true, "calls": true, "call": true, "throughput": true, "rps": true, "qps": true,
		"traffico": true, "traffic": true, "hits": true, "volume": true}
	quantityRe = phrases(`quant[oiae]`, `qual e`, `quale e`, `quali sono`, `qual`, `what is`, `what s`, `whats`, `what are`,
		`how (many|much|fast|slow|long|quick)`, `dimmi (la|il|i|le|quant\w*)`, `tell me (the|how)`, `voglio sapere`,
		`vorrei sapere`, `mi dici`, `sapere`, `misura`, `measure`, `calcola`, `calculate`, `compute`, `in media`, `on average`)
	statWords = map[string]string{
		"media": statAvg, "medio": statAvg, "medi": statAvg, "medie": statAvg, "average": statAvg, "avg": statAvg,
		"mean": statAvg, "mediana": statP50, "median": statP50, "p50": statP50, "p95": statP95, "p99": statP99,
		"massimo": statMax, "massima": statMax, "massimi": statMax, "max": statMax, "peggiore": statMax,
		"peggior": statMax, "worst": statMax, "maximum": statMax, "mediani": statP50, "mediane": statP50,
		"mediano": statP50, "slowest": statMax, "lentissima": statMax,
	}
	slowestRe = phrases(`piu lent\w*`, `slowest`, `most slow`, `la peggiore`, `il peggiore`)
	// Metric names that already ask for a number: "error rate", "throughput".
	numberMetricRe = phrases(`error rate`, `tas+\w* di er\w*`, `percentuale di err\w*`, `error count`, `numero di (errori|richieste|chiamate)`,
		`throughput`, `rps`, `qps`, `requests per (second|minute)`, `richieste al (secondo|minuto)`, `request count`,
		`volume di richieste`, `failure rate`, `tasso di fallimento`)
	methodUpperRe   = regexp.MustCompile(`\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\b`)
	methodContextRe = regexp.MustCompile(`\b(?:le|delle|tutte le|all|chiamate|richieste|endpoint|the|sulle|per le|di|only|solo) (get|post|put|patch|delete)\b|\b(get|post|put|patch|delete) (?:requests?|calls?|endpoints?|chiamate|richieste|api)\b`)
	pathRe          = regexp.MustCompile(`(?:^|\s)(/[a-z0-9_\-./{}:]+)`)
	notOnlyRe       = phrases(`non solo`, `not just`, `not only`)
	allEndpointsRe  = phrases(`tutti gli endpoint`, `tutte le chiamate`, `all endpoints`, `all calls`, `ogni endpoint`, `every endpoint`)
)

// measureParts extracts whatever measure details the message states, even if
// they are not enough on their own to make it a measure question.
func measureParts(prompt, plain string) Measure {
	var m Measure
	toks := tokens(plain)
	latency, errors, requests := false, false, false
	for _, tok := range toks {
		switch {
		case percentileRe.MatchString(tok) || hasAnyPrefix(tok, measureLatencyStems) || tok == "ms" || tok == "lat" || fuzzyIn(tok, []string{"latenza", "latency", "velocita"}):
			latency = true
		case statusCodeRe.MatchString(tok) || hasAnyPrefix(tok, errorStems) || fuzzyIn(tok, []string{"errori", "errore", "errors"}):
			errors = true
		case measureRequestWords[tok]:
			requests = true
		}
		if s, ok := statWords[tok]; ok && m.Stat == "" {
			m.Stat = s
		}
	}
	if responseRe.MatchString(plain) || strings.Contains(plain, "tasso di risposta") {
		latency = true
	}
	switch {
	case latency:
		m.Metric = metricLatency
	case errors:
		m.Metric = metricErrors
	case requests:
		m.Metric = metricRequests
	}
	if strings.Contains(plain, "in media") || strings.Contains(plain, "on average") {
		m.Stat = statAvg
	}
	if slowestRe.MatchString(plain) && m.Stat == "" {
		m.Stat = statMax
		if m.Metric == "" {
			m.Metric = metricLatency
		}
	}
	if numberMetricRe.MatchString(plain) && m.Metric == "" {
		m.Metric = metricErrors
		if phrases(`throughput`, `rps`, `qps`, `requests per`, `richieste al`, `request count`, `volume di richieste`, `numero di (richieste|chiamate)`).MatchString(plain) {
			m.Metric = metricRequests
		}
	}
	// A percentile alone is a latency question: "p95 overall today".
	if m.Metric == "" && (m.Stat == statP50 || m.Stat == statP95 || m.Stat == statP99) {
		m.Metric = metricLatency
	}
	if mm := methodUpperRe.FindStringSubmatch(prompt); mm != nil {
		m.Method = mm[1]
	} else if mm := methodContextRe.FindStringSubmatch(plain); mm != nil {
		m.Method = strings.ToUpper(mm[1] + mm[2])
	}
	if pm := pathRe.FindStringSubmatch(strings.ToLower(prompt)); pm != nil && len(pm[1]) > 1 && len(pm[1]) <= 100 {
		m.Path = strings.TrimRight(pm[1], ".,:?")
	}
	return m
}

// parseMeasure reports whether the message asks for a number rather than for
// "what is wrong": a metric plus a statistic, a quantity question or a filter.
func parseMeasure(prompt, plain string) (Measure, bool) {
	m := measureParts(prompt, plain)
	// "log di errore" and "non solo latenza" are not requests for a number.
	if m.Metric == "" || causeRe.MatchString(plain) || detectFocus(plain) == focusLogs || notOnlyRe.MatchString(plain) {
		return m, false
	}
	return m, m.Stat != "" || quantityRe.MatchString(plain) || m.Method != "" || m.Path != "" || numberMetricRe.MatchString(plain)
}

// mergeMeasure applies what a follow-up states on top of the previous measure.
func mergeMeasure(prev, cur Measure, plain string) Measure {
	out := prev
	if cur.Metric != "" {
		out.Metric = cur.Metric
	}
	if cur.Stat != "" {
		out.Stat = cur.Stat
	}
	if cur.Method != "" {
		out.Method = cur.Method
	}
	if cur.Path != "" {
		out.Path = cur.Path
	}
	if allEndpointsRe.MatchString(plain) {
		out.Method, out.Path = "", ""
	}
	return out
}

var validMetric = map[string]bool{metricLatency: true, metricErrors: true, metricRequests: true}
var validStat = map[string]bool{"": true, statAvg: true, statP50: true, statP95: true, statP99: true, statMax: true}
var validMethod = map[string]bool{"": true, "GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true}

func (m *Measure) sanitize() *Measure {
	if m == nil || !validMetric[m.Metric] || !validStat[m.Stat] || !validMethod[m.Method] {
		return nil
	}
	out := *m
	if len(out.Path) > 100 || (out.Path != "" && !strings.HasPrefix(out.Path, "/")) {
		out.Path = ""
	}
	return &out
}

// ----------------------------------------------------------------- answer

func (m Measure) matches(endpoint string) bool {
	if m.Method != "" && !regexp.MustCompile(`(^|\s)`+m.Method+`(\s|$)`).MatchString(strings.ToUpper(endpoint)) {
		return false
	}
	return m.Path == "" || strings.Contains(strings.ToLower(endpoint), m.Path)
}

// subject names what is measured: "le chiamate GET di **catalog-service**".
func (p phrasing) subject(m Measure, service string) string {
	var s string
	switch {
	case m.Method != "" && m.Path != "":
		s = fmt.Sprintf(p.pick("le chiamate %s a `%s`", "the %s calls to `%s`"), m.Method, m.Path)
	case m.Method != "":
		s = fmt.Sprintf(p.pick("le chiamate %s", "the %s calls"), m.Method)
	case m.Path != "":
		s = fmt.Sprintf(p.pick("le chiamate a `%s`", "the calls to `%s`"), m.Path)
	case service == "":
		return p.pick("tutti gli endpoint", "all endpoints")
	default:
		s = p.pick("gli endpoint", "the endpoints")
	}
	return s + p.of(service)
}

func (r *Runner) measureReport(ctx context.Context, scope Scope, m Measure, t texts, p phrasing, detail bool) *Report {
	subject := p.subject(m, scope.Service)
	steps := []string{fmt.Sprintf(p.pick("Interpreto la richiesta: %s di %s, %s", "Reading the request: %s of %s, %s"),
		p.metricName(m), stripMarkdown(subject), strings.ToLower(p.window(scope)))}
	out := &Context{From: scope.From, To: scope.To, Service: scope.Service, Today: scope.Today, Measure: &m}
	if r.Metrics == nil {
		return &Report{Answer: p.pick("Le metriche non sono disponibili.", "Metrics are not available."), Steps: steps, Context: out}
	}
	req := metrics.DashboardRequest{From: scope.From, To: scope.To, ServiceName: scope.Service}
	lat, err := r.Metrics.EndpointLatencies(ctx, req, endpointStatsLimit)
	if err != nil {
		return &Report{Answer: p.pick("Non sono riuscito a leggere le statistiche degli endpoint.", "I couldn't read the endpoint statistics."), Steps: steps, Context: out}
	}
	var errs []metrics.ErrorHotspot
	if m.Metric == metricErrors {
		if errs, err = r.Metrics.EndpointErrors(ctx, req, endpointStatsLimit); err != nil {
			return &Report{Answer: p.pick("Non sono riuscito a leggere gli errori degli endpoint.", "I couldn't read the endpoint errors."), Steps: steps, Context: out}
		}
	}
	lat = filterLatencies(lat, m)
	errs = filterErrors(errs, m)
	steps = append(steps, fmt.Sprintf(p.pick("Leggo le statistiche di %d endpoint", "Reading the statistics of %d endpoints"), len(lat)))
	if m.Metric == metricLatency && (m.Stat == "" || m.Stat == statAvg) {
		steps = append(steps, p.pick("Calcolo la media pesata sul numero di richieste", "Computing the average weighted by request count"))
	}

	var answer string
	var items []ContextItem
	switch m.Metric {
	case metricErrors:
		answer, items = p.errorsAnswer(scope, subject, lat, errs, detail)
	case metricRequests:
		answer, items = p.requestsAnswer(scope, subject, lat, detail)
	default:
		answer, items = p.latencyAnswer(scope, subject, m, lat, detail)
	}
	out.Items = items
	sugs := suggestions{}
	if !detail && len(lat) > 1 {
		d := p.detailsSuggestion()
		sugs.add(d.Label, d.Prompt)
	}
	sugs.add(p.pick("Cosa non va qui?", "What's wrong here?"), p.windowPrompt(scope.Service, scope.window())+p.pick(" problemi", " problems"))
	if m.Metric != metricLatency {
		sugs.add(p.pick("E la latenza?", "And latency?"), p.pick("e la latenza media?", "and the average latency?"))
	}
	if m.Metric != metricErrors {
		sugs.add(p.pick("E gli errori?", "And errors?"), p.pick("e quanti errori?", "and how many errors?"))
	}
	return &Report{Answer: answer, Steps: steps, Suggestions: sugs, Context: out}
}

func (p phrasing) metricName(m Measure) string {
	switch m.Metric {
	case metricErrors:
		return p.pick("errori", "errors")
	case metricRequests:
		return p.pick("richieste", "requests")
	}
	switch m.Stat {
	case statP50:
		return p.pick("latenza mediana", "median latency")
	case statP95, statP99:
		return p.pick("latenza ", "latency ") + m.Stat
	case statMax:
		return p.pick("latenza massima", "maximum latency")
	}
	return p.pick("latenza media", "average latency")
}

func stripMarkdown(s string) string { return strings.NewReplacer("**", "", "`", "").Replace(s) }

func filterLatencies(in []metrics.EndpointLatency, m Measure) []metrics.EndpointLatency {
	out := []metrics.EndpointLatency{}
	for _, e := range in {
		if e.Count > 0 && m.matches(e.Endpoint) {
			out = append(out, e)
		}
	}
	return out
}

func filterErrors(in []metrics.ErrorHotspot, m Measure) []metrics.ErrorHotspot {
	out := []metrics.ErrorHotspot{}
	for _, e := range in {
		if m.matches(e.Endpoint) {
			out = append(out, e)
		}
	}
	return out
}

func (p phrasing) noRequests(scope Scope, subject string) string {
	return fmt.Sprintf(p.pick("%s non trovo richieste per %s.", "%s I find no requests for %s."), p.window(scope), subject)
}

func (p phrasing) latencyAnswer(scope Scope, subject string, m Measure, lat []metrics.EndpointLatency, detail bool) (string, []ContextItem) {
	if len(lat) == 0 {
		return p.noRequests(scope, subject), nil
	}
	sort.SliceStable(lat, func(i, j int) bool { return lat[i].P95 > lat[j].P95 })
	var total int64
	var weighted float64
	for _, e := range lat {
		total += e.Count
		weighted += e.AvgMs * float64(e.Count)
	}
	avg := weighted / math.Max(float64(total), 1)
	slow := lat[0]
	var b strings.Builder
	win := p.window(scope)
	switch m.Stat {
	case statP50, statP95, statP99:
		pick := func(e metrics.EndpointLatency) float64 {
			switch m.Stat {
			case statP50:
				return e.P50
			case statP99:
				return e.P99
			}
			return e.P95
		}
		sort.SliceStable(lat, func(i, j int) bool { return pick(lat[i]) > pick(lat[j]) })
		slow = lat[0]
		if len(lat) == 1 {
			fmt.Fprintf(&b, p.pick("%s il %s di `%s` è **%s**.", "%s the %s of `%s` is **%s**."), win, m.Stat, shortEndpoint(slow.Endpoint), p.ms(pick(slow)))
		} else {
			fmt.Fprintf(&b, p.pick("%s il %s di %s va da %s a **%s**; il più alto è `%s`.", "%s the %s of %s ranges from %s to **%s**; the highest is `%s`."),
				win, m.Stat, subject, p.ms(pick(lat[len(lat)-1])), p.ms(pick(slow)), shortEndpoint(slow.Endpoint))
		}
	case statMax:
		sort.SliceStable(lat, func(i, j int) bool { return lat[i].P99 > lat[j].P99 })
		slow = lat[0]
		fmt.Fprintf(&b, p.pick("%s la chiamata più lenta tra %s è `%s`: p99 **%s**, media %s.", "%s the slowest of %s is `%s`: p99 **%s**, average %s."),
			win, subject, shortEndpoint(slow.Endpoint), p.ms(slow.P99), p.ms(slow.AvgMs))
	default:
		fmt.Fprintf(&b, p.pick("%s %s rispondono in media in **%s** (%d endpoint, %s richieste).", "%s, %s take **%s** on average (%d endpoints, %s requests)."),
			win, subject, p.ms(avg), len(lat), p.count(total))
		if len(lat) > 1 {
			fmt.Fprintf(&b, p.pick(" La più lenta è `%s`: media %s, p95 %s.", " The slowest is `%s`: average %s, p95 %s."),
				shortEndpoint(slow.Endpoint), p.ms(slow.AvgMs), p.ms(slow.P95))
		}
	}
	items := []ContextItem{{Kind: KindLatency, Service: slow.Service, Endpoint: slow.Endpoint}}
	if detail {
		b.WriteString("\n")
		items = nil
		for i, e := range lat {
			if i == measureDetailRows {
				fmt.Fprintf(&b, p.pick("…e altri %d.\n", "…and %d more.\n"), len(lat)-measureDetailRows)
				break
			}
			fmt.Fprintf(&b, p.pick("%d. **%s** `%s`: media %s, p95 %s, %s richieste\n", "%d. **%s** `%s`: average %s, p95 %s, %s requests\n"),
				i+1, e.Service, shortEndpoint(e.Endpoint), p.ms(e.AvgMs), p.ms(e.P95), p.count(e.Count))
			items = append(items, ContextItem{Kind: KindLatency, Service: e.Service, Endpoint: e.Endpoint})
		}
	}
	return strings.TrimSpace(b.String()), items
}

func (p phrasing) errorsAnswer(scope Scope, subject string, lat []metrics.EndpointLatency, errs []metrics.ErrorHotspot, detail bool) (string, []ContextItem) {
	var requests, errors int64
	for _, e := range lat {
		requests += e.Count
	}
	for _, e := range errs {
		errors += e.ErrorCount
	}
	if requests == 0 && errors == 0 {
		return p.noRequests(scope, subject), nil
	}
	requests = max(requests, errors)
	win := p.window(scope)
	var b strings.Builder
	if errors == 0 {
		fmt.Fprintf(&b, p.pick("✅ %s %s non hanno avuto errori (%s richieste).", "✅ %s, %s had no errors (%s requests)."), win, subject, p.count(requests))
		return b.String(), nil
	}
	sort.SliceStable(errs, func(i, j int) bool { return errs[i].ErrorCount > errs[j].ErrorCount })
	fmt.Fprintf(&b, p.pick("%s %s hanno avuto **%s errori** su %s richieste (**%s**).", "%s, %s had **%s errors** out of %s requests (**%s**)."),
		win, subject, p.count(errors), p.count(requests), p.pct(float64(errors)/float64(requests)*100))
	top := errs[0]
	if len(errs) > 1 {
		fmt.Fprintf(&b, p.pick(" Il più colpito è `%s` (**%s**) con %s errori.", " The most affected is `%s` (**%s**) with %s errors."),
			shortEndpoint(top.Endpoint), top.Service, p.count(top.ErrorCount))
	}
	items := []ContextItem{{Kind: KindHotspot, Service: top.Service, Endpoint: top.Endpoint}}
	if detail {
		b.WriteString("\n")
		items = nil
		for i, e := range errs {
			if i == measureDetailRows {
				fmt.Fprintf(&b, p.pick("…e altri %d.\n", "…and %d more.\n"), len(errs)-measureDetailRows)
				break
			}
			fmt.Fprintf(&b, p.pick("%d. **%s** `%s`: %s errori su %s (%s)\n", "%d. **%s** `%s`: %s errors out of %s (%s)\n"),
				i+1, e.Service, shortEndpoint(e.Endpoint), p.count(e.ErrorCount), p.count(e.TotalCount), p.pct(e.ErrorRate))
			items = append(items, ContextItem{Kind: KindHotspot, Service: e.Service, Endpoint: e.Endpoint})
		}
	}
	return strings.TrimSpace(b.String()), items
}

func (p phrasing) requestsAnswer(scope Scope, subject string, lat []metrics.EndpointLatency, detail bool) (string, []ContextItem) {
	if len(lat) == 0 {
		return p.noRequests(scope, subject), nil
	}
	sort.SliceStable(lat, func(i, j int) bool { return lat[i].Count > lat[j].Count })
	var total int64
	for _, e := range lat {
		total += e.Count
	}
	perMinute := float64(total) / math.Max(scope.window().Minutes(), 1)
	var b strings.Builder
	fmt.Fprintf(&b, p.pick("%s %s hanno ricevuto **%s richieste** (%s al minuto).", "%s, %s received **%s requests** (%s per minute)."),
		p.window(scope), subject, p.count(total), p.count(int64(math.Round(perMinute))))
	top := lat[0]
	if len(lat) > 1 {
		fmt.Fprintf(&b, p.pick(" La più chiamata è `%s` (%s).", " The busiest is `%s` (%s)."), shortEndpoint(top.Endpoint), p.count(top.Count))
	}
	items := []ContextItem{{Kind: KindLatency, Service: top.Service, Endpoint: top.Endpoint}}
	if detail {
		b.WriteString("\n")
		items = nil
		for i, e := range lat {
			if i == measureDetailRows {
				fmt.Fprintf(&b, p.pick("…e altri %d.\n", "…and %d more.\n"), len(lat)-measureDetailRows)
				break
			}
			fmt.Fprintf(&b, p.pick("%d. **%s** `%s`: %s richieste\n", "%d. **%s** `%s`: %s requests\n"), i+1, e.Service, shortEndpoint(e.Endpoint), p.count(e.Count))
			items = append(items, ContextItem{Kind: KindLatency, Service: e.Service, Endpoint: e.Endpoint})
		}
	}
	return strings.TrimSpace(b.String()), items
}
