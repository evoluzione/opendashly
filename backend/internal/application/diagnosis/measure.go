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

	routeStatsLimit   = 200
	measureDetailRows = 15
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
	// routeWords maps words for common route segments to the segment:
	// "la ricerca del catalogo" measures /search. Service names stay services.
	routeWords = map[string]string{"ricerca": "search", "ricerche": "search", "search": "search",
		"prodotti": "products", "prodotto": "products", "products": "products", "ordini": "orders",
		"ordine": "orders", "orders": "orders", "disponibilita": "availability", "availability": "availability",
		"giacenze": "stock", "stock": "stock", "articoli": "items", "items": "items", "autorizzazione": "authorize",
		"authorize": "authorize", "authorization": "authorize", "prenotazione": "reserve", "reserve": "reserve",
		"sessione": "session", "session": "session", "sessions": "session", "tracking": "tracking",
		"tracciamento": "tracking", "preventivo": "quote", "preventivi": "quote", "quote": "quote"}
	notOnlyRe      = phrases(`non solo`, `not just`, `not only`)
	allEndpointsRe = phrases(`tutti gli endpoint`, `tutte le chiamate`, `all endpoints`, `all calls`, `ogni endpoint`, `every endpoint`)
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
	if m.Path == "" {
		for _, tok := range toks {
			if seg, ok := routeWords[tok]; ok {
				m.Path = "/" + seg
				break
			}
		}
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
	steps := []string{fmt.Sprintf(p.pick("Interpreto la richiesta: %s %s, %s", "Reading the request: %s of %s, %s"),
		p.metricName(m), stripMarkdown(p.di(subject)), strings.ToLower(p.window(scope)))}
	out := &Context{From: scope.From, To: scope.To, Service: scope.Service, Today: scope.Today, Yesterday: scope.Yesterday, Measure: &m}
	if r.Metrics == nil {
		return &Report{Answer: p.pick("Le metriche non sono disponibili.", "Metrics are not available."), Steps: steps, Context: out}
	}
	filter := metrics.RouteFilter{Service: scope.Service, Method: m.Method, Path: m.Path}
	total, routes, err := r.Metrics.RouteStatsFor(ctx, scope.From, scope.To, filter, routeStatsLimit)
	if err != nil {
		return &Report{Answer: p.pick("Non sono riuscito a leggere le statistiche degli endpoint: il sistema è sotto carico, riprova tra poco.",
			"I couldn't read the endpoint statistics: the system is under load, try again shortly."), Steps: steps, Context: out}
	}
	steps = append(steps, fmt.Sprintf(p.pick("Aggrego %s richieste su %d rotte", "Aggregating %s requests over %d routes"), p.count(total.Count), len(routes)))

	var answer string
	var items []ContextItem
	switch m.Metric {
	case metricErrors:
		answer, items = p.errorsAnswer(scope, subject, total, routes, detail)
	case metricRequests:
		answer, items = p.requestsAnswer(scope, subject, total, routes, detail)
	default:
		answer, items = p.latencyAnswer(scope, subject, m, total, routes, detail)
	}
	out.Items = items
	sugs := suggestions{}
	if total.Count == 0 {
		// Nothing in this window: offer windows that may have data.
		if !scope.Today {
			sugs.add(p.pick("Oggi", "Today"), p.pick("e oggi?", "and today?"))
		}
		sugs.add(p.pick("Ultime 24 ore", "Last 24 hours"), p.pick("e nelle ultime 24 ore?", "and in the last 24 hours?"))
	}
	if !detail && len(routes) > 1 {
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

// di joins "di" with the article of an Italian subject: "di le chiamate" ->
// "delle chiamate", "di gli endpoint" -> "degli endpoint". English keeps the
// subject as is (the templates say "of %s").
func (p phrasing) di(subject string) string {
	if !p.it {
		return subject
	}
	switch {
	case strings.HasPrefix(subject, "le "):
		return "delle " + strings.TrimPrefix(subject, "le ")
	case strings.HasPrefix(subject, "gli "):
		return "degli " + strings.TrimPrefix(subject, "gli ")
	}
	return "di " + subject
}

func stripMarkdown(s string) string { return strings.NewReplacer("**", "", "`", "").Replace(s) }

func (p phrasing) noRequests(scope Scope, subject string) string {
	return fmt.Sprintf(p.pick("%s non trovo richieste per %s. Prova con un altro periodo.", "%s I find no requests for %s. Try another window."), p.window(scope), subject)
}

// approx marks percentiles, which are estimated from 100 ms-wide histogram buckets.
func (p phrasing) approx(ms float64) string { return "≈" + p.ms(ms) }

func percentileOf(r metrics.RouteStats, stat string) float64 {
	switch stat {
	case statP50:
		return r.P50
	case statP99:
		return r.P99
	}
	return r.P95
}

func routeItems(routes []metrics.RouteStats, kind string, n int) []ContextItem {
	out := []ContextItem{}
	for i, r := range routes {
		if i == n {
			break
		}
		out = append(out, ContextItem{Kind: kind, Service: r.Service, Endpoint: r.Route})
	}
	return out
}

func (p phrasing) routeList(b *strings.Builder, routes []metrics.RouteStats, line func(metrics.RouteStats) string) {
	b.WriteString("\n")
	for i, r := range routes {
		if i == measureDetailRows {
			fmt.Fprintf(b, p.pick("…e altre %d rotte.\n", "…and %d more routes.\n"), len(routes)-measureDetailRows)
			break
		}
		fmt.Fprintf(b, "%d. **%s** `%s`: %s\n", i+1, r.Service, r.Route, line(r))
	}
}

func (p phrasing) latencyAnswer(scope Scope, subject string, m Measure, total metrics.RouteStats, routes []metrics.RouteStats, detail bool) (string, []ContextItem) {
	if total.Count == 0 || len(routes) == 0 {
		return p.noRequests(scope, subject), nil
	}
	win := p.window(scope)
	var b strings.Builder
	switch m.Stat {
	case statP50, statP95, statP99:
		sort.SliceStable(routes, func(i, j int) bool { return percentileOf(routes[i], m.Stat) > percentileOf(routes[j], m.Stat) })
		fmt.Fprintf(&b, p.pick("%s il %s %s è **%s** (%s richieste).", "%s the %s of %s is **%s** (%s requests)."),
			win, m.Stat, p.di(subject), p.approx(percentileOf(total, m.Stat)), p.count(total.Count))
		if len(routes) > 1 {
			fmt.Fprintf(&b, p.pick(" Il più alto è `%s` (%s).", " The highest is `%s` (%s)."), routes[0].Route, p.approx(percentileOf(routes[0], m.Stat)))
		}
	case statMax:
		sort.SliceStable(routes, func(i, j int) bool { return routes[i].P99 > routes[j].P99 })
		fmt.Fprintf(&b, p.pick("%s la rotta più lenta tra %s è `%s`: p99 **%s**, media %s.", "%s the slowest route of %s is `%s`: p99 **%s**, average %s."),
			win, subject, routes[0].Route, p.approx(routes[0].P99), p.ms(routes[0].AvgMs))
	default:
		sort.SliceStable(routes, func(i, j int) bool { return routes[i].AvgMs > routes[j].AvgMs })
		fmt.Fprintf(&b, p.pick("%s %s rispondono in media in **%s** (%s richieste, p95 %s).", "%s, %s take **%s** on average (%s requests, p95 %s)."),
			win, subject, p.ms(total.AvgMs), p.count(total.Count), p.approx(total.P95))
		if len(routes) > 1 {
			fmt.Fprintf(&b, p.pick(" La più lenta è `%s`: media %s.", " The slowest is `%s`: average %s."), routes[0].Route, p.ms(routes[0].AvgMs))
		}
	}
	items := routeItems(routes, KindLatency, 1)
	if detail {
		p.routeList(&b, routes, func(r metrics.RouteStats) string {
			return fmt.Sprintf(p.pick("media %s, p95 %s, %s richieste", "average %s, p95 %s, %s requests"), p.ms(r.AvgMs), p.approx(r.P95), p.count(r.Count))
		})
		items = routeItems(routes, KindLatency, measureDetailRows)
	}
	return strings.TrimSpace(b.String()), items
}

func (p phrasing) errorsAnswer(scope Scope, subject string, total metrics.RouteStats, routes []metrics.RouteStats, detail bool) (string, []ContextItem) {
	if total.Count == 0 {
		return p.noRequests(scope, subject), nil
	}
	win := p.window(scope)
	var b strings.Builder
	if total.Errors == 0 {
		fmt.Fprintf(&b, p.pick("✅ %s %s non hanno restituito errori (%s richieste). Gli errori nelle chiamate verso altri servizi non sono contati qui.",
			"✅ %s, %s returned no errors (%s requests). Errors in calls to other services are not counted here."), win, subject, p.count(total.Count))
		return b.String(), nil
	}
	sort.SliceStable(routes, func(i, j int) bool { return routes[i].Errors > routes[j].Errors })
	fmt.Fprintf(&b, p.pick("%s %s hanno restituito **%s errori** su %s richieste (**%s**).", "%s, %s returned **%s errors** out of %s requests (**%s**)."),
		win, subject, p.count(total.Errors), p.count(total.Count), p.pct(float64(total.Errors)/float64(total.Count)*100))
	if len(routes) > 1 && routes[0].Errors > 0 {
		fmt.Fprintf(&b, p.pick(" Il più colpito è `%s` (**%s**) con %s errori.", " The most affected is `%s` (**%s**) with %s errors."),
			routes[0].Route, routes[0].Service, p.count(routes[0].Errors))
	}
	items := routeItems(routes, KindHotspot, 1)
	if detail {
		p.routeList(&b, routes, func(r metrics.RouteStats) string {
			return fmt.Sprintf(p.pick("%s errori su %s", "%s errors out of %s"), p.count(r.Errors), p.count(r.Count))
		})
		items = routeItems(routes, KindHotspot, measureDetailRows)
	}
	return strings.TrimSpace(b.String()), items
}

func (p phrasing) requestsAnswer(scope Scope, subject string, total metrics.RouteStats, routes []metrics.RouteStats, detail bool) (string, []ContextItem) {
	if total.Count == 0 {
		return p.noRequests(scope, subject), nil
	}
	sort.SliceStable(routes, func(i, j int) bool { return routes[i].Count > routes[j].Count })
	perMinute := float64(total.Count) / math.Max(scope.window().Minutes(), 1)
	var b strings.Builder
	fmt.Fprintf(&b, p.pick("%s %s hanno ricevuto **%s richieste** (%s al minuto).", "%s, %s received **%s requests** (%s per minute)."),
		p.window(scope), subject, p.count(total.Count), p.count(int64(math.Round(perMinute))))
	if len(routes) > 1 {
		fmt.Fprintf(&b, p.pick(" La più chiamata è `%s` (%s).", " The busiest is `%s` (%s)."), routes[0].Route, p.count(routes[0].Count))
	}
	items := routeItems(routes, KindLatency, 1)
	if detail {
		p.routeList(&b, routes, func(r metrics.RouteStats) string {
			return fmt.Sprintf(p.pick("%s richieste", "%s requests"), p.count(r.Count))
		})
		items = routeItems(routes, KindLatency, measureDetailRows)
	}
	return strings.TrimSpace(b.String()), items
}
