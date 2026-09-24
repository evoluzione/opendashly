package diagnosis

import (
	"fmt"
	"strings"
	"time"
)

// Link points at the data behind an answer: the UI turns it into a search
// page URL with the filters already set (in the viewer's time zone) and into
// a CSV/JSON download of the same rows.
type Link struct {
	Label      string    `json:"label"`
	Kind       string    `json:"kind"` // logs | traces | trace
	Service    string    `json:"service,omitempty"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	Severity   string    `json:"severity,omitempty"`
	ErrorsOnly bool      `json:"errorsOnly,omitempty"`
	Search     string    `json:"search,omitempty"`
	TraceID    string    `json:"traceId,omitempty"`
}

const (
	linkLogs   = "logs"
	linkTraces = "traces"
	linkTrace  = "trace"

	errorSeverities = "ERROR,FATAL"
)

var (
	// Asking for the data itself: a link to the search page or a download.
	linkRequestRe = phrases(`link`, `links`, `url`, `apri\w* (nella|la|in) ricerca`, `portami`, `vai (ai|alle|alla|al)`,
		`open (it |them )?in (the )?(search|explorer)`, `take me to`, `scarica\w*`, `download\w*`, `esporta\w*`,
		`export\w*`, `csv`, `json`, `excel`, `xlsx`, `pdf`, `salva\w*`, `save (it|them|the)`)
	logWordsRe   = phrases(`log`, `logs`, `messaggi`, `messages`)
	allDataRe    = phrases(`tutti i log`, `tutte le trace`, `tutti i dati`, `all (the )?logs`, `all (the )?traces`, `all (the )?data`, `anche quelli ok`, `non solo (gli )?errori`)
	traceWordsRe = phrases(`trace`, `traces`, `tracce`, `traccia`, `span`, `spans`)
)

func windowLink(kind, label string, scope Scope) Link {
	return Link{Label: label, Kind: kind, Service: scope.Service, From: scope.From, To: scope.To}
}

func (p phrasing) errorLogsLink(scope Scope) Link {
	l := windowLink(linkLogs, p.pick("Log di errore", "Error logs"), scope)
	l.Severity = errorSeverities
	return l
}

func (p phrasing) logsLink(scope Scope) Link {
	return windowLink(linkLogs, p.pick("Log", "Logs"), scope)
}

func (p phrasing) errorTracesLink(scope Scope, search string) Link {
	l := windowLink(linkTraces, p.pick("Trace in errore", "Error traces"), scope)
	l.ErrorsOnly, l.Search = true, search
	return l
}

func (p phrasing) tracesLink(scope Scope, search string) Link {
	l := windowLink(linkTraces, p.pick("Trace", "Traces"), scope)
	l.Search = search
	return l
}

func (p phrasing) traceLink(traceID string) Link {
	return Link{Label: p.pick("Trace ", "Trace ") + shortID(traceID), Kind: linkTrace, TraceID: traceID}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8] + "…"
	}
	return id
}

// focusLinks are the links that fit an analysis with that focus, only to data
// the analysis actually found (a link to an empty page is no help).
func (p phrasing) focusLinks(scope Scope, focus string, hasErrorLogs, hasErrorTraces bool) []Link {
	switch focus {
	case focusLatency, focusTraffic:
		return []Link{p.tracesLink(scope, "")}
	}
	links := []Link{}
	if hasErrorLogs {
		links = append(links, p.errorLogsLink(scope))
	}
	if hasErrorTraces && focus != focusLogs {
		links = append(links, p.errorTracesLink(scope, ""))
	}
	return links
}

// measureLinks open the rows a measure was computed on.
func (p phrasing) measureLinks(scope Scope, m Measure) []Link {
	search := m.Path
	if search == "" {
		search = m.Method
	}
	if m.Metric == metricErrors {
		return []Link{p.errorTracesLink(scope, search), p.errorLogsLink(scope)}
	}
	return []Link{p.tracesLink(scope, search)}
}

// linksReport answers "scaricami i log", "dammi il link" with the data links
// for the scope, errors only when the conversation is about errors.
func (p phrasing) linksReport(req request, prev *Context) *Report {
	scope := req.scope
	if scope.TraceID != "" {
		return &Report{Answer: p.pick("Ecco la trace: aprila nella ricerca o scaricala in JSON.", "Here is the trace: open it in the search or download it as JSON."),
			Links: []Link{p.traceLink(scope.TraceID)}, Context: prev}
	}
	// Errors only when the conversation is about errors or a general check,
	// unless the user asks for everything ("tutti i log", "all traces").
	answered := prev != nil && prev.Pending == ""
	measured := answered && prev.Measure != nil && prev.Measure.Metric != metricErrors
	diagnosed := answered && !measured && (prev.Focus == focusGeneral || prev.Focus == focusErrors || prev.Focus == focusLogs)
	errorsOnly := req.linkErrors || req.focus == focusErrors || diagnosed
	if allDataRe.MatchString(strings.ToLower(req.rawPlain)) {
		errorsOnly = false
	}
	wantLogs, wantTraces := req.linkLogs, req.linkTraces
	if !wantLogs && !wantTraces {
		wantLogs, wantTraces = true, true
	}
	var links []Link
	if wantLogs {
		if errorsOnly {
			links = append(links, p.errorLogsLink(scope))
		} else {
			links = append(links, p.logsLink(scope))
		}
	}
	if wantTraces {
		if errorsOnly {
			links = append(links, p.errorTracesLink(scope, ""))
		} else {
			links = append(links, p.tracesLink(scope, ""))
		}
	}
	answer := fmt.Sprintf(p.pick("Ecco i dati%s (%s): aprili nella ricerca con i filtri già impostati o scaricali in CSV.",
		"Here is the data%s (%s): open it in the search with the filters already set or download it as CSV."),
		p.where(scope.Service), strings.ToLower(p.window(scope)))
	out := &Context{From: scope.From, To: scope.To, Service: scope.Service, Focus: req.focus, Today: scope.Today, Yesterday: scope.Yesterday}
	if prev != nil && prev.Pending == "" {
		out.Items, out.Measure = prev.Items, prev.Measure
	}
	return &Report{Answer: answer, Links: links, Context: out}
}

// rows gives each numbered item of an answer its own actions: analyze it
// (the prompt resolveRef reads) and open the rows it was computed on.
func (p phrasing) rows(scope Scope, items []ContextItem) []Row {
	out := make([]Row, 0, len(items))
	for i, it := range items {
		s := scope
		if it.Service != "" {
			s.Service = it.Service
		}
		search := endpointSearch(it.Endpoint)
		var l Link
		switch {
		case it.TraceID != "":
			l = p.traceLink(it.TraceID)
		case it.Kind == KindLogPattern:
			l = p.errorLogsLink(s)
		case it.Kind == KindLatency || it.Kind == KindTrafficDrop:
			l = p.tracesLink(s, search)
		default:
			l = p.errorTracesLink(s, search)
		}
		out = append(out, Row{Prompt: fmt.Sprintf(p.pick("analizza il %d", "analyze #%d"), i+1), Link: &l})
	}
	return out
}

// endpointSearch turns "GET /orders/123/items" into "/orders/", the part
// shared by every call to the route, so the search finds all of them.
func endpointSearch(endpoint string) string {
	ep := shortEndpoint(endpoint)
	if i := strings.Index(ep, " "); i >= 0 {
		ep = ep[i+1:]
	}
	if i := strings.Index(ep, "…"); i >= 0 {
		ep = ep[:i]
	}
	return ep
}
