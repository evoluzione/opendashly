package diagnosis

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
)

// This file writes the short answers the chat shows by default: one sentence
// with the verdict and at most three numbered lines. The full report
// (report.go) is only produced when the user asks for details.

const conciseItems = 3

type phrasing struct{ it bool }

func phrasingFor(locale string) phrasing {
	return phrasing{it: !strings.HasPrefix(strings.ToLower(locale), "en")}
}

func (p phrasing) pick(it, en string) string {
	if p.it {
		return it
	}
	return en
}

// ---------------------------------------------------------------- numbers

func (p phrasing) count(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	sep := p.pick(".", ",")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func (p phrasing) decimal(f float64, digits int) string {
	s := strconv.FormatFloat(f, 'f', digits, 64)
	if p.it {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

func (p phrasing) pct(f float64) string {
	digits := 2
	if f >= 10 {
		digits = 1
	}
	return p.decimal(f, digits) + "%"
}

func (p phrasing) ms(f float64) string {
	if f >= 1000 {
		return p.decimal(f/1000, 1) + " s"
	}
	return strconv.FormatFloat(math.Round(f), 'f', 0, 64) + " ms"
}

// --------------------------------------------------------------- phrases

// window turns a scope into "Nell'ultima ora" / "In the last 3 hours".
func (p phrasing) window(s Scope) string {
	if s.Today {
		return p.pick("Oggi", "Today")
	}
	if s.Yesterday {
		return p.pick("Ieri", "Yesterday")
	}
	m := int(math.Round(s.window().Minutes()))
	switch {
	case m == 60:
		return p.pick("Nell'ultima ora", "In the last hour")
	case m < 60 || m%60 != 0:
		if m > 180 {
			return p.pick("Nelle ultime ", "In the last ") + p.decimal(float64(m)/60, 1) + p.pick(" ore", " hours")
		}
		return fmt.Sprintf(p.pick("Negli ultimi %d minuti", "In the last %d minutes"), m)
	case m == 7*24*60:
		return p.pick("Nell'ultima settimana", "In the last week")
	case m%(24*60) == 0 && m > 24*60:
		return fmt.Sprintf(p.pick("Negli ultimi %d giorni", "In the last %d days"), m/(24*60))
	}
	return fmt.Sprintf(p.pick("Nelle ultime %d ore", "In the last %d hours"), m/60)
}

// windowPrompt writes a request that parseScope understands, for buttons.
func (p phrasing) windowPrompt(service string, d time.Duration) string {
	phrase := fmt.Sprintf(p.pick("ultimi %d minuti", "last %d minutes"), int(d.Round(time.Minute).Minutes()))
	if service == "" {
		return phrase
	}
	return service + " " + phrase
}

func (p phrasing) problems(n int) string {
	if n == 1 {
		return p.pick("**1 problema**", "**1 problem**")
	}
	return fmt.Sprintf(p.pick("**%d problemi**", "**%d problems**"), n)
}

func (p phrasing) reply(in intent) string {
	switch in {
	case intentHelp:
		return p.pick(
			"Trovo cosa non va nella telemetria: confronto un periodo con quello precedente, raggruppo i log di errore e risalgo le trace fino all'origine dell'errore. Rispondo anche a domande precise, come la latenza media delle GET di un servizio.\n\nDimmi un servizio, un periodo o incollami un trace id.",
			"I find what is wrong in your telemetry: I compare a window with the previous one, group error logs and walk traces back to where the error started. I also answer precise questions, like the average latency of a service's GET endpoints.\n\nTell me a service, a time window or paste a trace id.")
	case intentGreeting:
		return p.pick("Ciao! Cosa controllo?", "Hi! What should I check?")
	case intentThanks:
		return p.pick("Figurati! Se vuoi continuo da qui:", "You're welcome! I can continue from here:")
	}
	return p.pick(
		"Non ho capito la richiesta. Posso dirti cosa non va (errori, latenza, traffico, log), darti un numero preciso o analizzare una trace: prova con `payment ultime 2 ore` o `latenza media delle GET del catalogo`.",
		"I did not understand the request. I can tell you what is wrong (errors, latency, traffic, logs), give you a precise number or analyze a trace: try `payment last 2 hours` or `average latency of the catalog GET endpoints`.")
}

// noItemFor answers a reference to an item about a service the list lacks.
func (p phrasing) noItemFor(service string) string {
	return fmt.Sprintf(p.pick("Nell'ultima risposta non c'è un punto su **%s**. Vuoi che lo analizzi?", "The last answer has no item about **%s**. Should I analyze it?"), service)
}

func (p phrasing) noItem(n int) string {
	if n == 0 {
		return p.pick("Nell'ultima risposta non c'erano punti da aprire.", "The last answer had no items to open.")
	}
	return fmt.Sprintf(p.pick("Nell'ultima risposta ci sono solo %d punti: dimmi quale aprire.", "The last answer has only %d items: tell me which one to open."), n)
}

var (
	unsupportedIT = map[string]string{
		unsupportedChart:    "Non so disegnare grafici: per quelli c'è la dashboard. Posso però darti i numeri qui.",
		unsupportedAction:   "Non posso fare azioni sui servizi (riavvii, cancellazioni, deploy, rollback): leggo solo la telemetria.",
		unsupportedAlert:    "Non posso creare o configurare alert. Posso però dirti se adesso c'è qualcosa che non va.",
		unsupportedExport:   "Non posso esportare file. Posso riassumerti i dati qui in chat.",
		unsupportedBusiness: "Non ho dati di business (utenti, vendite, fatturato): vedo solo trace e log dei servizi.",
	}
	unsupportedEN = map[string]string{
		unsupportedChart:    "I can't draw charts: the dashboard has those. I can give you the numbers here.",
		unsupportedAction:   "I can't act on services (restarts, deletes, deploys, rollbacks): I only read telemetry.",
		unsupportedAlert:    "I can't create or configure alerts. I can tell you whether something is wrong right now.",
		unsupportedExport:   "I can't export files. I can summarize the data here in the chat.",
		unsupportedBusiness: "I have no business data (users, sales, revenue): I only see service traces and logs.",
	}
)

// Texts used directly by Run.
func (p phrasing) unsupportedText(reason string) string {
	if p.it {
		return unsupportedIT[reason]
	}
	return unsupportedEN[reason]
}

func (p phrasing) noContextText() string {
	return p.pick(
		"Non so a cosa ti riferisci: in questa chat non ho ancora mostrato risultati. Chiedimi prima un'analisi, per esempio `ultima ora`.",
		"I don't know what you are referring to: I haven't shown any results in this chat yet. Ask me for an analysis first, for example `last hour`.")
}

func (p phrasing) infraNoteText() string {
	return p.pick("Non raccolgo metriche di CPU e memoria: ti dico cosa vedo da trace e log.",
		"I don't collect CPU or memory metrics: here is what traces and logs show.")
}

func (p phrasing) stepOpenItemText() string {
	return p.pick("Riprendo il punto %d della risposta precedente", "Picking up item %d of the previous answer")
}

// ------------------------------------------------------------- one-liners

var (
	versionSegRe = regexp.MustCompile(`^v\d+$`)
	httpCodeRe   = regexp.MustCompile(`\bHTTP (\d{3})\b`)
	exceptionRe  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.$]*(Error|Exception|Failure|Fault|Timeout|Declined|Refused|Denied|Unavailable)?$`)
)

// shortEndpoint hides ids in paths: "GET /inventory/stock/SKU-7503" -> "GET /inventory/stock/…".
func shortEndpoint(ep string) string {
	parts := strings.Split(strings.TrimPrefix(ep, "HTTP "), "/")
	for i, seg := range parts {
		if i > 0 && strings.ContainsAny(seg, "0123456789") && !versionSegRe.MatchString(seg) {
			parts[i] = "…"
		}
	}
	return strings.Join(parts, "/")
}

// shortError keeps the exception type and the HTTP status of a span error.
func shortError(detail string) string {
	parts := strings.Split(detail, ": ")
	out := []string{}
	if len(parts) > 0 && exceptionRe.MatchString(parts[0]) && !strings.HasPrefix(parts[0], "HTTP") {
		out = append(out, parts[0])
	}
	if m := httpCodeRe.FindStringSubmatch(detail); m != nil {
		out = append(out, "HTTP "+m[1])
	}
	if len(out) == 0 {
		if len(detail) > 60 {
			return detail[:60] + "…"
		}
		return detail
	}
	return strings.Join(out, ", ")
}

// line writes one finding on one line. scoped is the service the user asked
// about: a problem found in another service is labeled as called by it.
func (p phrasing) line(f Finding, scoped string) string {
	icon := map[string]string{SeverityCritical: "🔴", SeverityWarning: "🟠", SeverityInfo: "🔵"}[f.Severity]
	who := ""
	switch {
	case f.Service != "" && scoped != "" && f.Service != scoped:
		who = "**" + f.Service + "**" + fmt.Sprintf(p.pick(" (chiamato da %s): ", " (called by %s): "), scoped)
	case f.Service != "":
		who = "**" + f.Service + "**: "
	}
	before := func(format string, v string) string {
		if f.NoBaseline {
			return ""
		}
		return fmt.Sprintf(format, v)
	}
	var text string
	switch f.Kind {
	case KindRootCause:
		text = who + fmt.Sprintf(p.pick("errore %s su `%s`", "%s error on `%s`"), shortError(f.Detail), shortEndpoint(f.Endpoint))
	case KindHotspot:
		text = who + fmt.Sprintf(p.pick("%s errori su `%s`", "%s errors on `%s`"), p.count(int64(f.Current)), shortEndpoint(f.Endpoint)) +
			before(p.pick(" (prima %s)", " (was %s)"), p.count(int64(f.Baseline)))
	case KindLogPattern:
		pattern := f.Detail
		if len(pattern) > 70 {
			pattern = pattern[:70] + "…"
		}
		text = who + fmt.Sprintf(p.pick("«%s» ripetuto %s volte", "«%s» repeated %s times"), pattern, p.count(int64(f.Current))) +
			before(p.pick(" (prima %s)", " (was %s)"), p.count(int64(f.Baseline)))
	case KindLatency:
		text = who + fmt.Sprintf(p.pick("`%s` più lento, p95 %s (prima %s)", "`%s` slower, p95 %s (was %s)"), shortEndpoint(f.Endpoint), p.ms(f.Current), p.ms(f.Baseline))
	case KindErrorRate:
		text = fmt.Sprintf(p.pick("error rate al **%s**", "error rate at **%s**"), p.pct(f.Current)) + before(p.pick(" (prima %s)", " (was %s)"), p.pct(f.Baseline))
	case KindTrafficDrop:
		text = fmt.Sprintf(p.pick("traffico sceso a %s richieste (prima %s)", "traffic down to %s requests (was %s)"), p.count(int64(f.Current)), p.count(int64(f.Baseline)))
	case KindNoTraffic:
		text = fmt.Sprintf(p.pick("**nessuna richiesta** (prima %s)", "**no requests** (was %s)"), p.count(int64(f.Baseline)))
	}
	return icon + " " + text
}

// ----------------------------------------------------------- window answer

var focusKinds = map[string]map[string]bool{
	focusErrors:  {KindErrorRate: true, KindHotspot: true, KindRootCause: true, KindLogPattern: true, KindNoTraffic: true},
	focusLatency: {KindLatency: true},
	focusTraffic: {KindTrafficDrop: true, KindNoTraffic: true},
	focusLogs:    {KindLogPattern: true},
}

func (p phrasing) where(service string) string {
	if service == "" {
		return p.pick(" tra tutti i servizi", " across all services")
	}
	return p.pick(" su **", " on **") + service + "**"
}

func (p phrasing) of(service string) string {
	if service == "" {
		return ""
	}
	return p.pick(" di **", " of **") + service + "**"
}

// conciseWindow answers in one sentence plus at most three numbered lines.
func conciseWindow(p phrasing, scope Scope, focus string, a analysis) (string, []ContextItem) {
	selected := a.findings
	if kinds, ok := focusKinds[focus]; ok {
		selected = nil
		for _, f := range a.findings {
			if kinds[f.Kind] {
				selected = append(selected, f)
			}
		}
	}
	shown := selected
	if len(shown) > conciseItems {
		shown = shown[:conciseItems]
	}
	var b strings.Builder
	win := p.window(scope)
	cur := a.cur

	switch focus {
	case focusErrors:
		if cur != nil && cur.Satisfaction.ErrorRate < 0.01 && len(shown) > 0 && scope.Service != "" {
			fmt.Fprintf(&b, p.pick("%s le risposte di **%s** non hanno errori (%s richieste), ma alcune chiamate che fa verso altri servizi falliscono:",
				"%s **%s** answers without errors (%s requests), but some of the calls it makes to other services fail:"),
				win, scope.Service, p.count(cur.Satisfaction.Throughput.TotalRequests))
		} else if cur != nil {
			fmt.Fprintf(&b, p.pick("%s l'error rate%s è **%s** su %s richieste", "%s the error rate%s is **%s** over %s requests"),
				win, p.of(scope.Service), p.pct(cur.Satisfaction.ErrorRate), p.count(cur.Satisfaction.Throughput.TotalRequests))
			if a.base != nil && a.base.Satisfaction.Throughput.TotalRequests > 0 {
				fmt.Fprintf(&b, p.pick(" (prima %s)", " (was %s)"), p.pct(a.base.Satisfaction.ErrorRate))
			}
			b.WriteString(".")
			if len(shown) > 0 {
				b.WriteString(p.pick(" Da dove vengono:", " Where they come from:"))
			} else {
				b.WriteString(p.pick(" Nessuna fonte di errore anomala.", " No unusual error source."))
			}
		} else if len(shown) > 0 {
			fmt.Fprintf(&b, p.pick("%s ho trovato queste fonti di errore%s:", "%s I found these error sources%s:"), win, p.where(scope.Service))
		} else {
			fmt.Fprintf(&b, p.pick("%s non ho trovato fonti di errore anomale%s.", "%s I found no unusual error sources%s."), win, p.where(scope.Service))
		}
	case focusLatency:
		worst := worstEndpoint(cur)
		if worst != nil {
			fmt.Fprintf(&b, p.pick("%s l'endpoint più lento%s è `%s`, con p95 **%s** (media %s).", "%s the slowest endpoint%s is `%s`, with p95 **%s** (average %s)."),
				win, p.of(scope.Service), shortEndpoint(worst.Endpoint), p.ms(worst.P95), p.ms(worst.AvgMs))
		} else {
			b.WriteString(win + p.pick(" non ho dati di latenza", " I have no latency data") + p.of(scope.Service) + ".")
		}
		hasBase := a.base != nil && a.base.Satisfaction.Throughput.TotalRequests > 0
		switch {
		case len(shown) > 0:
			b.WriteString(p.pick(" Sono peggiorati:", " These got slower:"))
		case hasBase && (cur == nil || len(cur.Hotspots.SlowestEndpoints) <= 1):
			b.WriteString(p.pick(" Non è peggiorato rispetto al periodo precedente.", " It did not get slower than in the previous window."))
		case !hasBase && worst != nil:
			b.WriteString(p.pick(" Non ho dati del periodo precedente per dire se è peggiorato.", " I have no data for the previous window to tell whether it got slower."))
		}
		if len(shown) == 0 && cur != nil && len(cur.Hotspots.SlowestEndpoints) > 1 {
			if hasBase {
				b.WriteString(p.pick(" Nessun peggioramento rispetto al periodo precedente. I più lenti:", " No slowdown compared with the previous window. The slowest:"))
			} else {
				b.WriteString(p.pick(" I più lenti:", " The slowest:"))
			}
			items := []ContextItem{}
			b.WriteString("\n")
			for i, e := range cur.Hotspots.SlowestEndpoints {
				if i == conciseItems {
					break
				}
				fmt.Fprintf(&b, "%d. **%s**: `%s`, p95 %s\n", i+1, e.Service, shortEndpoint(e.Endpoint), p.ms(e.P95))
				items = append(items, ContextItem{Kind: KindLatency, Service: e.Service, Endpoint: e.Endpoint})
			}
			return strings.TrimSpace(b.String()), items
		}
	case focusTraffic:
		if cur != nil {
			fmt.Fprintf(&b, p.pick("%s sono arrivate **%s richieste**%s (%s al minuto)", "%s there were **%s requests**%s (%s per minute)"),
				win, p.count(cur.Satisfaction.Throughput.TotalRequests), p.where(scope.Service),
				p.count(int64(math.Round(float64(cur.Satisfaction.Throughput.TotalRequests)/math.Max(scope.window().Minutes(), 1)))))
			if a.base != nil && a.base.Satisfaction.Throughput.TotalRequests > 0 {
				prev := a.base.Satisfaction.Throughput.TotalRequests
				change := (float64(cur.Satisfaction.Throughput.TotalRequests) - float64(prev)) / float64(prev) * 100
				fmt.Fprintf(&b, p.pick(", %s rispetto a prima", ", %s compared with before"), signedPct(p, change))
			}
			b.WriteString(".")
		} else {
			b.WriteString(win + p.pick(" non ho dati di traffico.", " I have no traffic data."))
		}
		if len(shown) > 0 {
			b.WriteString(p.pick(" Da notare:", " Worth noting:"))
		}
	case focusLogs:
		if len(selected) == 0 {
			b.WriteString(win + p.pick(" non ci sono log di errore insoliti", " there are no unusual error logs") + p.of(scope.Service) + ".")
		} else {
			verb := p.pick("ci sono", "there are")
			if len(selected) == 1 {
				verb = p.pick("c'è", "there is")
			}
			fmt.Fprintf(&b, p.pick("%s %s %s nei log di errore%s:", "%s %s %s in the error logs%s:"), win, verb,
				pluralize(p, len(selected), "messaggio ricorrente", "messaggi ricorrenti", "recurring message", "recurring messages"), p.of(scope.Service))
		}
	default:
		if len(selected) == 0 && cur != nil && cur.Satisfaction.Throughput.TotalRequests == 0 && len(a.failed) == 0 {
			// No traffic at all is not "everything normal".
			fmt.Fprintf(&b, p.pick("%s non ci sono dati%s: il servizio non ha ricevuto richieste o la telemetria non era attiva. Prova con un altro periodo.",
				"%s there is no data%s: no requests were received or telemetry was off. Try another window."), win, p.where(scope.Service))
		} else if len(selected) == 0 && (len(a.failed) > 0 || a.metricsBusy) {
			// Some checks could not run: say so instead of "all good".
			fmt.Fprintf(&b, p.pick("%s nei dati che ho potuto leggere non vedo problemi%s.", "%s I see no problems in the data I could read%s."), win, p.where(scope.Service))
		} else if len(selected) == 0 {
			if cur != nil {
				fmt.Fprintf(&b, p.pick("✅ %s è tutto regolare%s: %s richieste, error rate %s, p95 peggiore %s.", "✅ %s everything looks normal%s: %s requests, %s error rate, worst p95 %s."),
					win, p.where(scope.Service), p.count(cur.Satisfaction.Throughput.TotalRequests), p.pct(cur.Satisfaction.ErrorRate), p.ms(worstP95(cur)))
			} else {
				b.WriteString("✅ " + win + p.pick(" non vedo problemi nei log e nelle trace", " I see no problems in logs and traces") + p.where(scope.Service) + ".")
			}
		} else {
			fmt.Fprintf(&b, p.pick("%s ho trovato %s%s:", "%s I found %s%s:"), win, p.problems(len(selected)), p.where(scope.Service))
		}
	}

	if len(shown) > 0 {
		b.WriteString("\n")
		for i, f := range shown {
			fmt.Fprintf(&b, "%d. %s\n", i+1, p.line(f, scope.Service))
		}
		if extra := len(selected) - len(shown); extra > 0 {
			fmt.Fprintf(&b, p.pick("…e altri %d.\n", "…and %d more.\n"), extra)
		}
	}
	if a.metricsBusy {
		b.WriteString(p.pick("\nLe metriche sono momentaneamente incomplete (sistema sotto carico): ho guardato solo log e trace, riprova tra poco per i numeri.",
			"\nMetrics are temporarily incomplete (system under load): I only looked at logs and traces, try again shortly for the numbers."))
	}
	if hasNoBaseline(shown) {
		b.WriteString(p.pick("\nNon ho dati del periodo precedente, quindi non posso dire cosa è cambiato.", "\nI have no data for the previous window, so I can't tell what changed."))
	}
	if len(a.failed) > 0 {
		fmt.Fprintf(&b, p.pick("\nNon sono riuscito a leggere: %s.", "\nI couldn't read: %s."), strings.Join(a.failed, ", "))
	}
	return strings.TrimSpace(b.String()), contextItems(shown, conciseItems)
}

func pluralize(p phrasing, n int, itOne, itMany, enOne, enMany string) string {
	if n == 1 {
		return "**1 " + p.pick(itOne, enOne) + "**"
	}
	return fmt.Sprintf("**%d %s**", n, p.pick(itMany, enMany))
}

func signedPct(p phrasing, change float64) string {
	sign := "+"
	if change < 0 {
		sign = "−"
	}
	return sign + p.decimal(math.Abs(change), 0) + "%"
}

func contextItems(findings []Finding, n int) []ContextItem {
	out := []ContextItem{}
	for i, f := range findings {
		if i == n {
			break
		}
		it := ContextItem{Kind: f.Kind, Service: f.Service, Endpoint: f.Endpoint}
		if len(f.TraceIDs) > 0 {
			it.TraceID = f.TraceIDs[0]
		}
		out = append(out, it)
	}
	return out
}

// ------------------------------------------------------------ trace answer

func conciseTrace(p phrasing, spans []query.TraceSpanEntry, logs []query.LogEntry, failed []string) string {
	if len(spans) == 0 {
		if len(failed) > 0 {
			return p.pick("Non sono riuscito a leggere questa trace.", "I couldn't read this trace.")
		}
		return p.pick("Non trovo questa trace: forse è scaduta o l'id è sbagliato.", "I can't find this trace: it may have expired or the id is wrong.")
	}
	var b strings.Builder
	if root, ok := rootCauseSpan(spans); ok {
		fmt.Fprintf(&b, p.pick("L'errore nasce in **%s**, su `%s`: %s.", "The error starts in **%s**, at `%s`: %s."),
			root.Service, shortEndpoint(root.Name), shortError(spanErrorDetail(root)))
	} else {
		b.WriteString(p.pick("In questa trace non ci sono errori.", "This trace has no errors."))
	}
	start, end := spans[0].StartTime, spans[0].StartTime
	services := map[string]bool{}
	for _, s := range spans {
		if s.StartTime.Before(start) {
			start = s.StartTime
		}
		if e := s.StartTime.Add(time.Duration(s.Duration)); e.After(end) {
			end = e
		}
		services[s.Service] = true
	}
	fmt.Fprintf(&b, p.pick(" La richiesta dura %s e attraversa %d servizi.", " The request takes %s and crosses %d services."), p.ms(float64(end.Sub(start))/1e6), len(services))
	if slow := slowestSpans(spans, 1); len(slow) == 1 {
		fmt.Fprintf(&b, p.pick(" Lo span più lento è `%s` (%s, %s).", " The slowest span is `%s` (%s, %s)."), shortEndpoint(slow[0].Name), slow[0].Service, p.ms(float64(slow[0].Duration)/1e6))
	}
	if len(logs) > 0 {
		fmt.Fprintf(&b, p.pick(" Ha %d log di errore collegati.", " It has %d linked error logs."), len(logs))
	}
	return b.String()
}

// ------------------------------------------------------------- suggestions

type suggestions []Suggestion

const maxSuggestions = 4

func (s *suggestions) add(label, prompt string) {
	if len(*s) >= maxSuggestions {
		return
	}
	for _, x := range *s {
		if x.Prompt == prompt {
			return
		}
	}
	*s = append(*s, Suggestion{Label: label, Prompt: prompt})
}

func (p phrasing) detailsSuggestion() Suggestion {
	return Suggestion{Label: p.pick("Dettagli", "Details"), Prompt: p.pick("dettagli", "details")}
}

func detailSuggestions(p phrasing) []Suggestion { return []Suggestion{p.detailsSuggestion()} }

// windowSuggestions offers the next moves: details, open the first item,
// focus on its service or back to all services, widen the window.
func windowSuggestions(p phrasing, scope Scope, items []ContextItem, detail bool) []Suggestion {
	out := suggestions{}
	if !detail && len(items) > 0 {
		d := p.detailsSuggestion()
		out.add(d.Label, d.Prompt)
	}
	if len(items) > 0 {
		out.add(p.pick("Analizza il n.1", "Analyze #1"), p.pick("analizza il primo", "analyze the first one"))
	}
	if scope.Service == "" {
		for _, it := range items {
			if it.Service != "" {
				out.add(p.pick("Solo ", "Only ")+it.Service, p.windowPrompt(it.Service, scope.window()))
				break
			}
		}
	} else {
		out.add(p.pick("Tutti i servizi", "All services"), p.windowPrompt("", scope.window()))
	}
	if scope.window() < 24*time.Hour {
		out.add(p.pick("Ultime 24 ore", "Last 24 hours"), p.windowPrompt(scope.Service, 24*time.Hour))
	}
	return out
}

func traceSuggestions(p phrasing, root query.TraceSpanEntry, hasRoot, detail bool) []Suggestion {
	out := suggestions{}
	if !detail {
		d := p.detailsSuggestion()
		out.add(d.Label, d.Prompt)
	}
	if hasRoot && root.Service != "" {
		out.add(p.pick("Solo ", "Only ")+root.Service, p.windowPrompt(root.Service, time.Hour))
	}
	out.add(p.pick("Panoramica ultima ora", "Last hour overview"), p.pick("ultima ora", "last hour"))
	return out
}

func starterSuggestions(p phrasing, in intent) []Suggestion {
	out := []Suggestion{
		{Label: p.pick("Ultima ora", "Last hour"), Prompt: p.pick("ultima ora", "last hour")},
		{Label: p.pick("Ultime 24 ore", "Last 24 hours"), Prompt: p.pick("ultime 24 ore", "last 24 hours")},
		{Label: p.pick("Errori di oggi", "Errors today"), Prompt: p.pick("errori di oggi", "errors today")},
	}
	if in != intentHelp {
		out = append(out, Suggestion{Label: p.pick("Cosa sai fare?", "What can you do?"), Prompt: p.pick("cosa sai fare?", "what can you do?")})
	}
	return out
}

func worstEndpoint(cur *metrics.DashboardResponse) *metrics.EndpointLatency {
	if cur == nil {
		return nil
	}
	var worst *metrics.EndpointLatency
	for i := range cur.Hotspots.SlowestEndpoints {
		if e := &cur.Hotspots.SlowestEndpoints[i]; worst == nil || e.P95 > worst.P95 {
			worst = e
		}
	}
	return worst
}

func worstP95(cur *metrics.DashboardResponse) float64 {
	if w := worstEndpoint(cur); w != nil {
		return w.P95
	}
	return 0
}

// ------------------------------------------------------------- questions

// askMissing returns the question to ask when an analysis lacks its time
// window or its service filter, or nil when everything is known. What was
// understood so far travels in the context and is completed by the answer.
func (p phrasing) askMissing(req request, services []string, prev *Context) *Report {
	// The previous answer's window and service are one click away.
	answered := prev != nil && prev.Pending == ""
	ctx := &Context{From: req.scope.From, To: req.scope.To, Today: req.scope.Today, Yesterday: req.scope.Yesterday, Service: req.scope.Service,
		Focus: req.focus, WindowSet: req.windowSet, ServiceSet: req.serviceSet, Detail: req.detail}
	if req.action == actMeasure {
		m := req.measure
		ctx.Measure = &m
	}
	var question string
	var sugs []Suggestion
	switch {
	case len(req.candidates) > 1:
		ctx.Pending, ctx.Candidates = slotService, req.candidates
		question = p.pick("Hai nominato più servizi: quale guardo?", "You named more than one service: which one should I look at?")
		for _, s := range req.candidates {
			sugs = append(sugs, Suggestion{Label: s, Prompt: s})
		}
		sugs = append(sugs, Suggestion{Label: p.pick("Tutti i servizi", "All services"), Prompt: p.pick("tutti i servizi", "all services")})
	case !req.windowSet:
		ctx.Pending = slotWindow
		question = p.pick("Su che periodo?", "Which time window?")
		if w := prev.scopeAtOrNil(req.scope.To); answered && w != nil && !isPresetWindow(*w) {
			prompt := p.windowPrompt("", w.window())
			if w.Today {
				prompt = p.pick("oggi", "today")
			}
			if w.Yesterday {
				prompt = p.pick("ieri", "yesterday")
			}
			sugs = append(sugs, Suggestion{Label: p.pick("Come prima (", "Same as before (") + strings.ToLower(p.window(*w)) + ")", Prompt: prompt})
		}
		sugs = append(sugs, []Suggestion{
			{Label: p.pick("Ultimi 15 minuti", "Last 15 minutes"), Prompt: p.pick("ultimi 15 minuti", "last 15 minutes")},
			{Label: p.pick("Ultima ora", "Last hour"), Prompt: p.pick("ultima ora", "last hour")},
			{Label: p.pick("Ultime 24 ore", "Last 24 hours"), Prompt: p.pick("ultime 24 ore", "last 24 hours")},
			{Label: p.pick("Oggi", "Today"), Prompt: p.pick("oggi", "today")},
			{Label: p.pick("Ultimi 7 giorni", "Last 7 days"), Prompt: p.pick("ultimi 7 giorni", "last 7 days")},
		}...)
	case !req.serviceSet:
		ctx.Pending = slotService
		question = p.pick("Su quali servizi? Tutti o uno in particolare?", "Which services? All of them or a specific one?")
		sugs = []Suggestion{{Label: p.pick("Tutti i servizi", "All services"), Prompt: p.pick("tutti i servizi", "all services")}}
		if answered && prev.Service != "" {
			sugs = append(sugs, Suggestion{Label: prev.Service, Prompt: prev.Service})
		}
		for _, s := range services {
			if len(sugs) == 7 {
				break
			}
			if answered && s == prev.Service {
				continue
			}
			sugs = append(sugs, Suggestion{Label: s, Prompt: s})
		}
	default:
		return nil
	}
	return &Report{Answer: p.understood(req) + " " + question, Suggestions: sugs, Context: ctx}
}

// understood restates the request so the user can check it before the analysis.
func (p phrasing) understood(req request) string {
	var what string
	if req.action == actMeasure {
		subject := p.subject(req.measure, "")
		switch req.measure.Metric {
		case metricErrors:
			what = p.pick("Ok, conto gli errori per ", "Ok, I'll count the errors for ") + subject
		case metricRequests:
			what = p.pick("Ok, conto le richieste per ", "Ok, I'll count the requests for ") + subject
		default:
			what = fmt.Sprintf(p.pick("Ok, calcolo la %s per %s", "Ok, I'll compute the %s for %s"), p.metricName(req.measure), subject)
		}
	} else {
		what = map[string]string{
			focusGeneral: p.pick("Ok, controllo cosa non va", "Ok, I'll check what is wrong"),
			focusErrors:  p.pick("Ok, analizzo gli errori", "Ok, I'll analyze the errors"),
			focusLatency: p.pick("Ok, analizzo la latenza", "Ok, I'll analyze latency"),
			focusTraffic: p.pick("Ok, analizzo il traffico", "Ok, I'll analyze traffic"),
			focusLogs:    p.pick("Ok, analizzo i log di errore", "Ok, I'll analyze the error logs"),
		}[req.focus]
	}
	if req.serviceSet {
		if req.scope.Service != "" {
			what += p.pick(" su **", " on **") + req.scope.Service + "**"
		} else {
			what += p.pick(" su tutti i servizi", " across all services")
		}
	}
	if req.windowSet {
		what += " (" + strings.ToLower(p.window(req.scope)) + ")"
	}
	return what + "."
}

// isPresetWindow reports whether a window is already one of the window buttons.
func isPresetWindow(s Scope) bool {
	if s.Today {
		return true
	}
	switch s.window() {
	case 15 * time.Minute, time.Hour, 24 * time.Hour, maxWindow:
		return true
	}
	return false
}
