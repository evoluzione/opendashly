package diagnosis

import (
	"fmt"
	"strings"
	"time"

	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
)

// maxFindings keeps the report scannable; the rest are summarized as a count.
const maxFindings = 6

type texts struct {
	title, scopeAll, scopeService, found, none, summary, advice, more, partial, degraded string
	checkMetrics, checkLogs, checkTraces                                                 string
	stepScope, stepMetrics, stepMetricsBad, stepMetricsOk, stepLogs                      string
	stepTraces, stepTracesNone, stepRank, stepTrace, stepTraceRoot, stepTraceSlow        string
	allServices, phraseMinutes                                                           string
	sugTrace, sugService, sugAll, sugWiden, sugNarrow, sugOverview                       string
	sugDay, sugToday, sugHelp, promptOverview, promptDay, promptToday                    string
	replyHelp, replyGreeting, replyThanks, replyUnclear, noBaseline                      string
	findingNoBase                                                                        map[string]string
	finding, hint                                                                        map[string]string
	severity                                                                             map[string]string
	traceTitle, traceNotFound, traceRoot, traceNoError, traceSlow, traceLogs             string
}

var textsIT = texts{
	title:          "## Diagnosi",
	scopeAll:       "Periodo **%s**, tutti i servizi, confrontato con %s.",
	scopeService:   "Periodo **%s**, servizio **%s**, confrontato con %s.",
	found:          "### Problemi trovati (%d)",
	none:           "Nessuna anomalia rispetto al periodo precedente.",
	summary:        "Richieste: **%d** · error rate **%.2f%%** · p95 peggiore **%.0f ms**",
	advice:         "### Cosa controllare",
	more:           "… e altri %d",
	partial:        "Controlli non completati: %s.",
	degraded:       "Metriche parziali (%s): i confronti potrebbero essere incompleti.",
	checkMetrics:   "metriche",
	checkLogs:      "log di errore",
	checkTraces:    "trace in errore",
	stepScope:      "Interpreto la richiesta: periodo %s, %s",
	stepMetrics:    "Leggo le metriche e le confronto con %s",
	stepMetricsBad: "Trovate %d anomalie nelle metriche: approfondisco",
	stepMetricsOk:  "Metriche stabili: controllo comunque log e trace",
	stepLogs:       "Leggo %d log di errore e li raggruppo per messaggio: %d da segnalare",
	stepTraces:     "Apro %d trace in errore e risalgo allo span che ha originato il problema",
	stepTracesNone: "Nessuna trace in errore da aprire",
	stepRank:       "Ordino %d problemi per gravità",
	stepTrace:      "Carico %d span e %d log di errore della trace",
	stepTraceRoot:  "Risalgo all'origine dell'errore: %s in %s",
	stepTraceSlow:  "Cerco gli span più lenti",
	allServices:    "tutti i servizi",
	phraseMinutes:  "ultimi %d minuti",
	sugTrace:       "Analizza la trace %s",
	sugService:     "Concentrati su %s",
	sugAll:         "Tutti i servizi",
	sugWiden:       "Allarga a 24 ore",
	sugNarrow:      "Solo ultimi 15 minuti",
	sugOverview:    "Panoramica ultima ora",
	sugDay:         "Ultime 24 ore",
	sugToday:       "Errori di oggi",
	sugHelp:        "Cosa sai fare?",
	promptOverview: "ultima ora",
	promptDay:      "ultime 24 ore",
	promptToday:    "errori di oggi",
	replyHelp: "Analizzo la telemetria e ti dico cosa non va, con le evidenze. In pratica:\n" +
		"- **confronto un periodo con quello precedente**: error rate, cali di traffico, latenza p95 per endpoint, endpoint con errori nuovi\n" +
		"- **raggruppo i log di errore** per messaggio e segnalo quelli nuovi o in crescita\n" +
		"- **apro le trace in errore** e trovo lo span dove l'errore è nato\n" +
		"- **analizzo una singola trace** se mi incolli il suo id\n\n" +
		"Scrivimi per esempio `payment-service ultimi 30 minuti`, `perché checkout è lento?` o `errori di oggi`.",
	replyGreeting: "Ciao! Dimmi cosa vuoi controllare: un servizio, un periodo o un trace id. Oppure parti da qui:",
	replyThanks:   "Figurati! Se vuoi continuo da qui:",
	replyUnclear:  "Non ho capito cosa vuoi che controlli. Posso analizzare un periodo, un servizio o una trace: prova con `payment-service ultime 2 ore` oppure scegli da qui.",
	noBaseline:    "Nel periodo di confronto non ci sono dati: mostro lo stato attuale senza confronto.",
	findingNoBase: map[string]string{
		KindErrorRate:  "Error rate al **%.2f%%**",
		KindHotspot:    "`%s`: **%.0f errori** su %d richieste",
		KindLogPattern: "Log di errore ripetuto **%.0f volte**: `%s`",
	},
	severity: map[string]string{SeverityCritical: "🔴", SeverityWarning: "🟠", SeverityInfo: "🔵"},
	finding: map[string]string{
		KindErrorRate:   "Error rate al **%.2f%%** (prima %.2f%%)",
		KindNoTraffic:   "**Nessuna richiesta** ricevuta (prima %.0f)",
		KindTrafficDrop: "Traffico calato a **%.0f** richieste (prima %.0f)",
		KindLatency:     "Latenza p95 di `%s` salita a **%.0f ms** (prima %.0f ms)",
		KindHotspot:     "`%s`: **%.0f errori** (prima %.0f)",
		KindLogPattern:  "Log di errore ripetuto **%.0f volte** (prima %.0f): `%s`",
		KindRootCause:   "Errore originato in `%s`: %s",
	},
	hint: map[string]string{
		KindErrorRate:   "Apri le trace in errore e verifica deploy o dipendenze cambiate nel periodo.",
		KindNoTraffic:   "Verifica che il servizio sia attivo e che il collector riceva dati.",
		KindTrafficDrop: "Controlla health check, bilanciatore e client a monte.",
		KindLatency:     "Confronta gli span lenti: query al database, chiamate esterne, lock.",
		KindHotspot:     "Filtra le trace per questo endpoint e guarda lo span in errore più profondo.",
		KindLogPattern:  "Cerca il messaggio nei log e apri una delle trace collegate.",
		KindRootCause:   "Parti dallo span indicato: è il punto in cui l'errore ha avuto origine.",
	},
	traceTitle:    "## Diagnosi trace `%s`",
	traceNotFound: "Trace non trovata o senza span.",
	traceRoot:     "**Origine dell'errore:** `%s` in **%s**: %s",
	traceNoError:  "Nessuno span in errore.",
	traceSlow:     "### Span più lenti",
	traceLogs:     "### Log di errore collegati",
}

var textsEN = texts{
	title:          "## Diagnosis",
	scopeAll:       "Window **%s**, all services, compared with %s.",
	scopeService:   "Window **%s**, service **%s**, compared with %s.",
	found:          "### Problems found (%d)",
	none:           "No anomalies compared with the previous window.",
	summary:        "Requests: **%d** · error rate **%.2f%%** · worst p95 **%.0f ms**",
	advice:         "### What to check",
	more:           "… and %d more",
	partial:        "Checks not completed: %s.",
	degraded:       "Partial metrics (%s): comparisons may be incomplete.",
	checkMetrics:   "metrics",
	checkLogs:      "error logs",
	checkTraces:    "error traces",
	stepScope:      "Reading the request: window %s, %s",
	stepMetrics:    "Reading metrics and comparing them with %s",
	stepMetricsBad: "Found %d anomalies in the metrics: digging deeper",
	stepMetricsOk:  "Metrics are stable: checking logs and traces anyway",
	stepLogs:       "Reading %d error logs and grouping them by message: %d worth reporting",
	stepTraces:     "Opening %d error traces and walking back to the span where the problem started",
	stepTracesNone: "No error traces to open",
	stepRank:       "Ranking %d problems by severity",
	stepTrace:      "Loading %d spans and %d error logs of the trace",
	stepTraceRoot:  "Walking back to the error origin: %s in %s",
	stepTraceSlow:  "Looking for the slowest spans",
	allServices:    "all services",
	phraseMinutes:  "last %d minutes",
	sugTrace:       "Analyze trace %s",
	sugService:     "Focus on %s",
	sugAll:         "All services",
	sugWiden:       "Widen to 24 hours",
	sugNarrow:      "Only last 15 minutes",
	sugOverview:    "Last hour overview",
	sugDay:         "Last 24 hours",
	sugToday:       "Errors today",
	sugHelp:        "What can you do?",
	promptOverview: "last hour",
	promptDay:      "last 24 hours",
	promptToday:    "errors today",
	replyHelp: "I analyze your telemetry and tell you what is wrong, with evidence. In practice:\n" +
		"- **I compare a window with the previous one**: error rate, traffic drops, p95 latency per endpoint, endpoints with new errors\n" +
		"- **I group error logs** by message and flag the new or growing ones\n" +
		"- **I open error traces** and find the span where the error started\n" +
		"- **I analyze a single trace** if you paste its id\n\n" +
		"Try for example `payment-service last 30 minutes`, `why is checkout slow?` or `errors today`.",
	replyGreeting: "Hi! Tell me what to check: a service, a time window or a trace id. Or start from here:",
	replyThanks:   "You're welcome! I can continue from here:",
	replyUnclear:  "I did not understand what you want me to check. I can analyze a time window, a service or a trace: try `payment-service last 2 hours` or pick one of these.",
	noBaseline:    "The comparison window has no data: showing the current state without comparison.",
	findingNoBase: map[string]string{
		KindErrorRate:  "Error rate at **%.2f%%**",
		KindHotspot:    "`%s`: **%.0f errors** out of %d requests",
		KindLogPattern: "Error log repeated **%.0f times**: `%s`",
	},
	severity: textsIT.severity,
	finding: map[string]string{
		KindErrorRate:   "Error rate at **%.2f%%** (was %.2f%%)",
		KindNoTraffic:   "**No requests** received (was %.0f)",
		KindTrafficDrop: "Traffic dropped to **%.0f** requests (was %.0f)",
		KindLatency:     "p95 latency of `%s` rose to **%.0f ms** (was %.0f ms)",
		KindHotspot:     "`%s`: **%.0f errors** (was %.0f)",
		KindLogPattern:  "Error log repeated **%.0f times** (was %.0f): `%s`",
		KindRootCause:   "Error originated in `%s`: %s",
	},
	hint: map[string]string{
		KindErrorRate:   "Open the error traces and check deploys or dependencies that changed in the window.",
		KindNoTraffic:   "Check that the service is up and the collector is receiving data.",
		KindTrafficDrop: "Check health checks, load balancer and upstream clients.",
		KindLatency:     "Compare the slow spans: database queries, external calls, locks.",
		KindHotspot:     "Filter traces by this endpoint and look at the deepest error span.",
		KindLogPattern:  "Search the message in logs and open one of the linked traces.",
		KindRootCause:   "Start from the span shown: it is where the error originated.",
	},
	traceTitle:    "## Trace diagnosis `%s`",
	traceNotFound: "Trace not found or without spans.",
	traceRoot:     "**Error origin:** `%s` in **%s**: %s",
	traceNoError:  "No errored spans.",
	traceSlow:     "### Slowest spans",
	traceLogs:     "### Linked error logs",
}

func textsFor(locale string) texts {
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return textsEN
	}
	return textsIT
}

func formatRange(from, to time.Time) string {
	layout := "15:04"
	if to.Sub(from) >= 24*time.Hour {
		layout = "02/01 15:04"
	}
	return from.UTC().Format(layout) + "–" + to.UTC().Format(layout) + " UTC"
}

func formatFinding(t texts, f Finding) string {
	format := t.finding[f.Kind]
	var line string
	if nb, ok := t.findingNoBase[f.Kind]; ok && f.NoBaseline {
		switch f.Kind {
		case KindErrorRate:
			line = fmt.Sprintf(nb, f.Current)
		case KindHotspot:
			line = fmt.Sprintf(nb, f.Endpoint, f.Current, f.Count)
		case KindLogPattern:
			line = fmt.Sprintf(nb, f.Current, f.Detail)
		}
		format = ""
	}
	switch f.Kind {
	case KindErrorRate, KindTrafficDrop:
		if format != "" {
			line = fmt.Sprintf(format, f.Current, f.Baseline)
		}
	case KindNoTraffic:
		line = fmt.Sprintf(format, f.Baseline)
	case KindLatency, KindHotspot:
		if format != "" {
			line = fmt.Sprintf(format, f.Endpoint, f.Current, f.Baseline)
		}
	case KindLogPattern:
		if format != "" {
			line = fmt.Sprintf(format, f.Current, f.Baseline, f.Detail)
		}
	case KindRootCause:
		line = fmt.Sprintf(format, f.Endpoint, f.Detail)
	}
	if f.Service != "" {
		line += " · " + f.Service
	}
	if len(f.TraceIDs) > 0 {
		ids := make([]string, len(f.TraceIDs))
		for i, id := range f.TraceIDs {
			ids[i] = "`" + id + "`"
		}
		line += " · trace " + strings.Join(ids, ", ")
	}
	return t.severity[f.Severity] + " " + line
}

func renderWindow(t texts, scope Scope, findings []Finding, cur *metrics.DashboardResponse, failed []string) string {
	var b strings.Builder
	baseFrom, baseTo := scope.baseline()
	b.WriteString(t.title + "\n\n")
	if scope.Service == "" {
		fmt.Fprintf(&b, t.scopeAll+"\n\n", formatRange(scope.From, scope.To), formatRange(baseFrom, baseTo))
	} else {
		fmt.Fprintf(&b, t.scopeService+"\n\n", formatRange(scope.From, scope.To), scope.Service, formatRange(baseFrom, baseTo))
	}
	if cur != nil {
		worst := 0.0
		for _, e := range cur.Hotspots.SlowestEndpoints {
			if e.P95 > worst {
				worst = e.P95
			}
		}
		fmt.Fprintf(&b, t.summary+"\n\n", cur.Satisfaction.Throughput.TotalRequests, cur.Satisfaction.ErrorRate, worst)
		if hasNoBaseline(findings) {
			b.WriteString(t.noBaseline + "\n\n")
		}
		if cur.Health.Status != "" && cur.Health.Status != "ok" {
			fmt.Fprintf(&b, t.degraded+"\n\n", cur.Health.Reason)
		}
	}
	if len(findings) == 0 {
		b.WriteString(t.none + "\n")
	} else {
		fmt.Fprintf(&b, t.found+"\n", len(findings))
		seen := map[string]bool{}
		hints := []string{}
		for i, f := range findings {
			if i == maxFindings {
				fmt.Fprintf(&b, t.more+"\n", len(findings)-maxFindings)
				break
			}
			fmt.Fprintf(&b, "%d. %s\n", i+1, formatFinding(t, f))
			if !seen[f.Kind] {
				seen[f.Kind] = true
				hints = append(hints, t.hint[f.Kind])
			}
		}
		b.WriteString("\n" + t.advice + "\n")
		for i, h := range hints {
			fmt.Fprintf(&b, "%d. %s\n", i+1, h)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(&b, "\n"+t.partial+"\n", strings.Join(failed, ", "))
	}
	return b.String()
}

func renderTrace(t texts, traceID string, spans []query.TraceSpanEntry, logs []query.LogEntry, failed []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, t.traceTitle+"\n\n", traceID)
	if len(spans) == 0 {
		b.WriteString(t.traceNotFound + "\n")
	} else {
		if root, ok := rootCauseSpan(spans); ok {
			fmt.Fprintf(&b, t.traceRoot+"\n\n", root.Name, root.Service, spanErrorDetail(root))
		} else {
			b.WriteString(t.traceNoError + "\n\n")
		}
		b.WriteString(t.traceSlow + "\n")
		for _, s := range slowestSpans(spans, criticalPathLen) {
			mark := ""
			if isErrorStatus(s.Status) {
				mark = " 🔴"
			}
			fmt.Fprintf(&b, "- `%s` · %s · **%.1f ms**%s\n", s.Name, s.Service, float64(s.Duration)/1e6, mark)
		}
	}
	if len(logs) > 0 {
		b.WriteString("\n" + t.traceLogs + "\n")
		for i, l := range logs {
			if i == criticalPathLen {
				break
			}
			body := strings.TrimSpace(l.Body)
			if len(body) > maxPatternLen {
				body = body[:maxPatternLen] + "…"
			}
			fmt.Fprintf(&b, "- `%s`\n", strings.ReplaceAll(body, "`", "'"))
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(&b, "\n"+t.partial+"\n", strings.Join(failed, ", "))
	}
	return b.String()
}

func serviceLabel(t texts, service string) string {
	if service == "" {
		return t.allServices
	}
	return service
}

func hasNoBaseline(findings []Finding) bool {
	for _, f := range findings {
		if f.NoBaseline {
			return true
		}
	}
	return false
}
