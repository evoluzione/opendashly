package diagnosis

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Context is what the previous answer was about. The UI sends it back with
// the next message so that follow-ups ("il primo", "dettagli", "e ieri?")
// refer to that answer. It comes from the client, so sanitize() validates it.
type Context struct {
	From    time.Time     `json:"from"`
	To      time.Time     `json:"to"`
	Service string        `json:"service,omitempty"`
	TraceID string        `json:"traceId,omitempty"`
	Focus   string        `json:"focus,omitempty"`
	Today   bool          `json:"today,omitempty"`
	Measure *Measure      `json:"measure,omitempty"`
	Items   []ContextItem `json:"items,omitempty"`
	// Pending is set when the assistant asked for a missing detail ("window"
	// or "service") before analyzing; the fields above hold what is known.
	Pending    string   `json:"pending,omitempty"`
	WindowSet  bool     `json:"windowSet,omitempty"`
	ServiceSet bool     `json:"serviceSet,omitempty"`
	Detail     bool     `json:"detail,omitempty"`
	Candidates []string `json:"candidates,omitempty"`
}

// ContextItem is one numbered line of the previous answer.
type ContextItem struct {
	Kind     string `json:"kind"`
	Service  string `json:"service,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	TraceID  string `json:"traceId,omitempty"`
}

const (
	focusGeneral = ""
	focusErrors  = "errors"
	focusLatency = "latency"
	focusTraffic = "traffic"
	focusLogs    = "logs"

	maxContextItems = 10
)

var validFocus = map[string]bool{focusGeneral: true, focusErrors: true, focusLatency: true, focusTraffic: true, focusLogs: true}

var validKinds = map[string]bool{KindErrorRate: true, KindTrafficDrop: true, KindNoTraffic: true, KindLatency: true,
	KindHotspot: true, KindLogPattern: true, KindRootCause: true}

var fullTraceIDRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// sanitize drops anything in a client-sent context that does not match the
// known services, trace id format, kinds and window limits.
func (c *Context) sanitize(services []string) *Context {
	if c == nil {
		return nil
	}
	known := map[string]bool{}
	for _, s := range services {
		known[s] = true
	}
	out := &Context{From: c.From, To: c.To, Today: c.Today, Measure: c.Measure.sanitize(),
		WindowSet: c.WindowSet, ServiceSet: c.ServiceSet, Detail: c.Detail}
	if c.Pending == slotWindow || c.Pending == slotService {
		out.Pending = c.Pending
	}
	for _, s := range c.Candidates {
		if known[s] && len(out.Candidates) < maxContextItems {
			out.Candidates = append(out.Candidates, s)
		}
	}
	if !out.To.After(out.From) || out.To.Sub(out.From) > maxWindow {
		out.To = out.From.Add(defaultWindow)
	}
	if known[c.Service] {
		out.Service = c.Service
	}
	if fullTraceIDRe.MatchString(c.TraceID) {
		out.TraceID = c.TraceID
	}
	if validFocus[c.Focus] {
		out.Focus = c.Focus
	}
	for _, it := range c.Items {
		if len(out.Items) == maxContextItems {
			break
		}
		if !validKinds[it.Kind] {
			continue
		}
		clean := ContextItem{Kind: it.Kind}
		if known[it.Service] {
			clean.Service = it.Service
		}
		if len(it.Endpoint) <= 200 {
			clean.Endpoint = it.Endpoint
		}
		if fullTraceIDRe.MatchString(it.TraceID) {
			clean.TraceID = it.TraceID
		}
		out.Items = append(out.Items, clean)
	}
	return out
}

// scopeAt re-anchors the previous window to now, keeping its length and service.
func (c *Context) scopeAt(now time.Time) Scope {
	d := c.To.Sub(c.From)
	if d <= 0 || d > maxWindow {
		d = defaultWindow
	}
	if c.Today {
		d = sinceMidnight(now)
	}
	return Scope{From: now.Add(-d), To: now, Service: c.Service, TraceID: c.TraceID, Explicit: true, WindowMentioned: true, Today: c.Today}
}

// ---------------------------------------------------------------- request

type action int

const (
	actNew          action = iota // analyze the scope stated in this message
	actOpenItem                   // drill into an item of the previous answer
	actDetails                    // full version of the previous answer
	actFollowUp                   // previous request with some parts changed
	actNoContextRef               // refers to a previous answer that does not exist
	actReply                      // help, greeting, thanks, unclear
	actUnsupported                // something the assistant cannot do
	actMeasure                    // a precise number ("latenza media delle GET")
	actNoMatch                    // refers to an item the previous answer does not have
)

type request struct {
	action      action
	intent      intent
	scope       Scope
	focus       string
	detail      bool
	ref         int // 1-based item index, -1 for the last one
	unsupported string
	infraNote   bool
	measure     Measure
	// windowSet / serviceSet tell whether the user stated them: an analysis
	// only starts once both are known (see Run).
	windowSet  bool
	serviceSet bool
	candidates []string
}

const (
	slotWindow  = "window"
	slotService = "service"
)

var (
	// Answers that leave a filter open: "tutti", "nessun filtro", "any".
	anyServiceRe = phrases(`tutti`, `tutte`, `tutto`, `all`, `qualsiasi`, `any`, `nessun filtro`, `nessuno`, `none`, `no filter`, `every`, `ognuno`, `entrambi`, `both`)
	// "fai tu", "non so": take the default for the question asked.
	defaultRe = phrases(`fai tu`, `fai te`, `decidi tu`, `decidi te`, `vedi tu`, `scegli tu`, `non so`, `boh`, `dunno`, `idk`, `you pick`, `pick one`, `your call`, `whatever you want`, `indifferente`, `come vuoi`, `default`, `whatever`, `you choose`, `up to you`, `dont know`, `don t know`, `no idea`, `standard`)
)

// understand decides what to do with a message given the previous answer.
func understand(prompt string, services []string, now time.Time, prev *Context) request {
	norm := normalize(prompt)
	plain := strings.TrimSpace(strings.ReplaceAll(norm, "?", " "))
	scope := parseScope(prompt, services, now)
	in := classify(prompt, scope)
	req := request{
		intent:    in,
		scope:     scope,
		focus:     detectFocus(norm),
		detail:    detailRe.MatchString(plain) || fuzzyDetail(plain),
		infraNote: infraRe.MatchString(plain),
	}
	hasItems := prev != nil && len(prev.Items) > 0
	ref, byContent := parseRef(plain, scope, prev)
	measure, isMeasure := parseMeasure(prompt, plain)
	measureFollowUp := prev != nil && prev.Measure != nil && ref == 0 && in != intentHelp && in != intentThanks &&
		(followUpStartRe.MatchString(plain) || len(tokens(plain)) <= 4) &&
		(measure.Metric != "" || measure.Stat != "" || measure.Method != "" || measure.Path != "" || scope.Mentions > 0 || scope.Explicit || allEndpointsRe.MatchString(plain))
	// An HTTP method in capitals ("le DELETE") is a filter, not a request to delete.
	unsupported := unsupportedReason(plain)
	if unsupported == unsupportedAction && measure.Method != "" && methodUpperRe.MatchString(prompt) {
		unsupported = ""
	}
	if !isMeasure && detectFocus(norm) == focusLogs {
		isMeasure = false
	}

	if scope.Mentions > 1 {
		req.candidates = findServices(pathRe.ReplaceAllString(norm, " "), services)
	}
	// "in generale come va?": a status question with its filter stated.
	if (in == intentGreeting || in == intentUnclear) && statusWordRe.MatchString(plain) && (allServicesRe.MatchString(plain) || scope.Explicit || scope.Mentions > 0) {
		in, req.intent = intentDiagnose, intentDiagnose
	}
	if prev != nil && prev.Pending != "" {
		if r, ok := fillPending(req, plain, now, prev); ok {
			return r
		}
		prev = nil // the message does not answer the question: treat it as new
		hasItems = false
	}
	req.serviceSet = scope.Mentions == 1 || allServicesRe.MatchString(plain)
	req.windowSet = scope.Explicit
	topic := req.focus != focusGeneral || req.detail || causeRe.MatchString(plain) || statusPhraseRe.MatchString(plain) || statusWordRe.MatchString(plain)
	selfContained := (scope.Mentions > 0 || allServicesRe.MatchString(plain)) && scope.Explicit && topic && !reuseRe.MatchString(plain)
	candidateFollowUp := prev != nil && in != intentHelp && in != intentThanks && in != intentGreeting &&
		!selfContained && (in == intentDiagnose || req.focus != focusGeneral || scope.Mentions > 0 || scope.Explicit ||
		rerunRe.MatchString(plain) || allServicesRe.MatchString(plain) || followUpStartRe.MatchString(plain))

	switch {
	case scope.TraceID != "":
		req.action = actNew
	case unsupported != "" && in != intentThanks:
		req.action, req.unsupported = actUnsupported, unsupported
	case ref != 0 || byContent:
		switch {
		case prev == nil:
			req.action = actNoContextRef
		case !hasItems || resolveRef(ref, prev.Items) < 0:
			req.action = actNoMatch
		default:
			req.action, req.ref = actOpenItem, ref
		}
	case measureFollowUp:
		req.action = actMeasure
		req.measure = mergeMeasure(*prev.Measure, measure, plain)
		req.scope = mergeScope(prev.scopeAt(now), scope, plain)
	case isMeasure && !(in == intentHelp && scope.Mentions == 0 && !scope.Explicit):
		req.action, req.measure = actMeasure, measure
		if prev != nil && !selfContained {
			req.scope = mergeScope(prev.scopeAt(now), scope, plain)
		}
	case req.detail && in != intentHelp && !thanksForDetailsRe.MatchString(plain) && scope.Mentions == 0 && !scope.WindowMentioned:
		switch {
		case prev == nil:
			req.action = actNoContextRef
		case (req.focus != focusGeneral && req.focus != prev.Focus) || resetsFocus(plain) || servicesAllRe.MatchString(plain):
			req.action = actFollowUp
			req.scope = mergeScope(prev.scopeAt(now), scope, plain)
			switch {
			case resetsFocus(plain):
				req.focus = focusGeneral
			case req.focus == focusGeneral:
				req.focus = prev.Focus
			}
		default:
			req.action = actDetails
		}
	case prev == nil && followUpStartRe.MatchString(plain) && in != intentDiagnose && scope.Mentions == 0 && !scope.Explicit:
		req.action = actNoContextRef
	case candidateFollowUp:
		req.action = actFollowUp
		req.scope = mergeScope(prev.scopeAt(now), scope, plain)
		switch {
		case resetsFocus(plain):
			req.focus = focusGeneral
		case req.focus == focusGeneral:
			req.focus = prev.Focus
		}
		// Nothing changed and the user asks for "all of it": the full answer.
		if req.scope.Service == prev.Service && req.focus == prev.Focus && !scope.Explicit && everythingRe.MatchString(plain) {
			req.action = actDetails
		}
	case in == intentDiagnose:
		req.action = actNew
	default:
		req.action = actReply
	}
	return req
}

// fillPending completes a request the assistant asked a question about: the
// message answers it when it states a window or service, leaves the filter
// open ("tutti") or delegates the choice ("fai tu").
func fillPending(req request, plain string, now time.Time, prev *Context) (request, bool) {
	scope := req.scope
	delegated := defaultRe.MatchString(plain)
	anyService := anyServiceRe.MatchString(plain) || allServicesRe.MatchString(plain)
	answers := scope.Explicit || scope.Mentions > 0 || anyService || delegated
	if !answers || req.intent == intentHelp {
		return req, false
	}
	out := request{intent: intentDiagnose, focus: prev.Focus, detail: prev.Detail || req.detail, infraNote: req.infraNote}
	out.action = actNew
	if prev.Measure != nil {
		out.action, out.measure = actMeasure, *prev.Measure
	}
	out.scope = Scope{From: now.Add(-defaultWindow), To: now}
	if prev.WindowSet {
		out.scope = prev.scopeAt(now)
	}
	out.windowSet = prev.WindowSet
	if scope.Explicit {
		out.scope.From, out.scope.To, out.scope.Today = scope.From, scope.To, scope.Today
		out.windowSet = true
	}
	out.scope.Explicit = true
	out.scope.Service, out.serviceSet = prev.Service, prev.ServiceSet
	switch {
	case scope.Mentions == 1:
		out.scope.Service, out.serviceSet = scope.Service, true
	case scope.Mentions > 1:
		out.candidates = req.candidates
	case anyService && (prev.Pending == slotService || !prev.ServiceSet):
		out.scope.Service, out.serviceSet = "", true
	}
	if delegated {
		switch prev.Pending {
		case slotWindow:
			out.windowSet = true
		case slotService:
			out.serviceSet = true
		}
	}
	return out, true
}

// ------------------------------------------------------------- references

var (
	ordinalWords = map[string]int{
		"primo": 1, "prima": 1, "first": 1, "1st": 1, "1o": 1, "secondo": 2, "seconda": 2, "second": 2,
		"2nd": 2, "2o": 2, "terzo": 3, "terza": 3, "third": 3, "3rd": 3, "3o": 3, "quarto": 4,
		"quarta": 4, "fourth": 4, "4th": 4, "quinto": 5, "quinta": 5, "fifth": 5, "5th": 5,
		"sesto": 6, "sesta": 6, "sixth": 6, "6th": 6, "ultimo": -1, "ultima": -1, "last": -1,
		"lultimo": -1, "lultima": -1, "penultimo": -2, "penultima": -2,
	}
	// Ordinals worth typo matching ("secndo", "prmo"); window words like
	// "ultimi" are deliberately left out.
	fuzzyOrdinals  = []string{"primo", "secondo", "seconda", "terzo", "terza", "quarto", "quinto", "second", "third", "fourth", "fifth"}
	spelledNumbers = map[string]int{"uno": 1, "one": 1, "due": 2, "two": 2, "tre": 3, "three": 3, "quattro": 4, "four": 4, "cinque": 5, "five": 5, "sei": 6, "six": 6}
	articles       = map[string]bool{"il": true, "la": true, "l": true, "lo": true, "the": true, "al": true,
		"alla": true, "sul": true, "sulla": true, "del": true, "della": true, "dal": true, "dalla": true, "nel": true, "nella": true}
	itemNouns = map[string]bool{"errore": true, "errori": true, "problema": true, "problemi": true,
		"trace": true, "traccia": true, "voce": true, "punto": true, "item": true, "one": true,
		"error": true, "problem": true, "issue": true, "risultato": true, "anomalia": true,
		"endpoint": true, "servizio": true, "service": true, "log": true, "pattern": true,
		"segnalazione": true, "result": true, "entry": true, "finding": true, "riga": true, "line": true}
	// Words after an ordinal that make it something else: "l ultima ora", "la prima volta".
	notItemAfter = map[string]bool{"ora": true, "ore": true, "hour": true, "hours": true, "minuti": true,
		"minutes": true, "giorno": true, "giorni": true, "day": true, "days": true, "settimana": true,
		"week": true, "mese": true, "month": true, "volta": true, "volte": true, "time": true,
		"times": true, "night": true, "notte": true, "deploy": true, "release": true, "rilascio": true,
		"thing": true, "cosa": true, "mattina": true, "morning": true, "sera": true}
	numberedRe = regexp.MustCompile(`\b(?:numero|num|nr|n|punto|voce|item|number|no|riga|line)\s*(\d|uno|one|due|two|tre|three|quattro|four|cinque|five|sei|six)\b|#\s*(\d)\b|\b(?:il|la|l|the|al|sul|del)\s+(\d)(?:\s|$)`)
	// "quello", "that one", "aprila", "open it": the first item of the previous answer.
	deicticRe = regexp.MustCompile(`\b(quell[oa]|quest[oa]) (errore|problema|trace|traccia|endpoint|log|pattern|voce)\b|` +
		`\b(analizza|apri|approfondisci|spiega|guarda|mostra|mostrami|open|analy[sz]e|check|explain|show me|dimmi di piu su|tell me more about|more on) (quello|quella|questo|questa|it|that|this|that one|this one)\b|` +
		`\b(that|this) (one|error|trace|issue|problem|endpoint)\b|` +
		`\b(aprila|aprilo|analizzalo|analizzala|guardalo|guardala|spiegamelo|spiegamela|approfondiscilo|approfondiscila)\b|` +
		`\b(apri|aprimi|mostrami|open|show me) (la|the) (trace|traccia)\b`)
)

// parseRef finds a reference to an item of the previous answer: the 1-based
// index (-1 = last, -2 = second to last), or byContent when the message picks
// an item by what it is about ("quello del carrello", "the slow one").
func parseRef(plain string, scope Scope, prev *Context) (ref int, byContent bool) {
	toks := tokens(plain)
	if secondToLastRe.MatchString(plain) {
		return -2, false
	}
	// A bare number right after a numbered list: "2".
	if len(toks) == 1 && len(toks[0]) == 1 && toks[0] >= "1" && toks[0] <= "9" && prev != nil && len(prev.Items) > 0 {
		return int(toks[0][0] - '0'), false
	}
	type hit struct{ n, pos int }
	var hits []hit
	for i, tok := range toks {
		next, prevTok := "", ""
		if i+1 < len(toks) {
			next = toks[i+1]
		}
		if i > 0 {
			prevTok = toks[i-1]
		}
		n, ok := ordinalWords[tok]
		if !ok && (articles[prevTok] || itemNouns[next]) && len(tok) >= 4 {
			for _, w := range fuzzyOrdinals {
				if withinEdits(tok, w, 1) {
					n, ok = ordinalWords[w], true
					break
				}
			}
		}
		if !ok || notItemAfter[next] || startsWithDigit(next) || (numberWords[next] != 0 && !itemNouns[next]) || tok == "quarto" && next == "d" {
			continue
		}
		if (tok == "secondo" || tok == "second") && (prevTok == "al" || prevTok == "per" || prevTok == "a") && !itemNouns[next] {
			continue // "richieste al secondo", "per second"
		}
		// "prima"/"first"/"last" alone mean "before", "first of all", "latest".
		if !articles[prevTok] && !itemNouns[next] && tok != "lultimo" && tok != "lultima" && next != "" {
			continue
		}
		if !articles[prevTok] && !itemNouns[next] && (tok == "prima" || tok == "first" || tok == "last") {
			continue
		}
		hits = append(hits, hit{n, i})
	}
	if len(hits) > 0 {
		// With several ordinals the one after an action verb wins ("the first
		// one was fine, check the second"); otherwise the last one mentioned.
		best := hits[len(hits)-1]
		for _, h := range hits {
			for j := max(0, h.pos-3); j < h.pos; j++ {
				if hasAnyPrefix(toks[j], actionStems) || hasAnyPrefix(toks[j], []string{"apri", "open", "analizz", "espand", "expand"}) {
					best = h
				}
			}
		}
		return best.n, false
	}
	if m := numberedRe.FindStringSubmatch(plain); m != nil {
		for _, g := range m[1:] {
			if g == "" {
				continue
			}
			if n, ok := spelledNumbers[g]; ok {
				return n, false
			}
			if n, _ := strconv.Atoi(g); n > 0 {
				return n, false
			}
		}
	}
	if prev != nil && len(prev.Items) > 0 && contentRefRe.MatchString(plain) {
		if idx := matchItem(plain, scope, prev.Items); idx >= 0 {
			return idx + 1, false
		}
		return 0, true
	}
	if deicticRe.MatchString(plain) {
		if prev != nil && traceWordRe.MatchString(plain) {
			for i, it := range prev.Items {
				if it.TraceID != "" {
					return i + 1, false
				}
			}
		}
		return 1, false
	}
	return 0, false
}

func startsWithDigit(s string) bool { return s != "" && s[0] >= '0' && s[0] <= '9' }

var (
	secondToLastRe = phrases(`penultim[oa]`, `second to last`, `second last`, `second-to-last`)
	traceWordRe    = phrases(`trace`, `traccia`)
	// "quello del carrello", "the slow one", "la trace di checkout": an item
	// picked by what it is about. Plural "quelli" narrows the scope instead.
	contentRefRe = regexp.MustCompile(`\b(quell[oa]|quest[oa]) (del|della|dello|dei|delle|di|su|sul|sulla|dell|con|che)\b|` +
		`\bthe [a-z0-9 -]{2,30} (one|trace|error|issue|endpoint)\b|` +
		`\b(la|della|nella|sulla|the) (trace|traccia) (di|del|della|dei|of|for|on)\b`)
	kindWords = map[string]string{
		"lento": KindLatency, "lenta": KindLatency, "slow": KindLatency, "latenza": KindLatency, "latency": KindLatency,
		"log": KindLogPattern, "logs": KindLogPattern, "messaggio": KindLogPattern, "message": KindLogPattern,
		"traffico": KindTrafficDrop, "traffic": KindTrafficDrop,
	}
)

// matchItem scores items against the message: the service it names, a kind
// word ("slow", "log") and words of the endpoint ("refresh token").
func matchItem(plain string, scope Scope, items []ContextItem) int {
	toks := tokens(plain)
	best, bestScore, tie := -1, 0, false
	for i, it := range items {
		score := 0
		if scope.Service != "" && it.Service == scope.Service {
			score += 3
		}
		ep := strings.ToLower(it.Endpoint)
		for _, tok := range toks {
			if k, ok := kindWords[tok]; ok && k == it.Kind {
				score += 2
			}
			if len(tok) >= 4 && !commonWords[tok] && !itemNouns[tok] && strings.Contains(ep, tok) {
				score += 2
			}
			if (tok == "trace" || tok == "traccia") && it.TraceID != "" {
				score++
			}
		}
		switch {
		case score > bestScore:
			best, bestScore, tie = i, score, false
		case score == bestScore && score > 0:
			tie = true
		}
	}
	if bestScore < 2 || tie {
		return -1
	}
	return best
}

// resolveRef maps a parsed reference to an index into items, or -1.
func resolveRef(ref int, items []ContextItem) int {
	switch {
	case ref < 0 && len(items) >= -ref:
		return len(items) + ref
	case ref >= 1 && ref <= len(items):
		return ref - 1
	}
	return -1
}

// ------------------------------------------------------ details, follow-ups

var (
	detailRe = phrases(`dettagli`, `details?`, `approfondisci`, `approfondimento`, `more info`,
		`piu info\w*`, `piu dettagli`, `dimmi di piu`, `tell me more`, `full report`, `report completo`,
		`tutto il report`, `espandi`, `expand`, `verbose`, `mostra(mi)? tutto`, `show (me )?(all|everything)`,
		`gli altri`, `le altre`, `the others`, `the rest`, `tutti i problemi`, `all (the )?problems`,
		`elenco completo`, `lista completa`, `in dettaglio`, `nel dettaglio`, `spiegami meglio`, `explain more`,
		`c e altro`, `ce altro`, `altro`, `anything else`, `what else`, `tutto qui`, `e tutto`, `versione completa`,
		`versione lunga`, `long version`, `full version`, `show me tutto`, `tutto tutto`, `the other ones`,
		`other ones`, `others`, `dimmi tutto`, `tell me everything`, `quadro completo`, `full picture`,
		`the rest`, `il resto`, `dettaglio`, `more detail`, `report dettagliato`, `analisi dettagliata`, `detailed report`,
		`detailed analysis`, `in modo dettagliato`, `resoconto dettagliato`)
	// "grazie per i dettagli" thanks for details already shown.
	// A status question states its own topic: "come va il catalogo", "stato del gateway".
	statusWordRe       = phrases(`come va`, `come sta`, `come stanno`, `come vanno`, `how s`, `hows`, `how is`, `how are`, `stato`, `status`, `situazione`, `salute`, `health`, `panoramica`, `overview`)
	thanksForDetailsRe = phrases(`(grazie|thanks|thank you|thx) (per|for) (i |the |tutti i )?(dettagli|details)`)
	followUpStartRe    = regexp.MustCompile(`^(e|and|what about|how about|invece|anche|also|same for|stessa cosa|idem|ripeti|repeat|do it again|rifallo|rifai|now|ora|poi|then|solo|only|just|e ora|and now|e adesso|adesso|e invece|e per|e su|e sul|e sulla|e con|e il|e la|e i|e le|e gli|e l)\b`)
	followUpEndRe      = regexp.MustCompile(`\b(invece|instead|anche|too|as well)$`)
	// servicesAllRe names all services explicitly; allServicesRe also accepts
	// looser words ("everything", "in generale") that can mean "the whole report".
	servicesAllRe = phrases(`tutti i servizi`, `all services`, `every service`, `each service`, `ogni servizio`, `tutto il sistema`, `whole system`, `entire system`, `intero sistema`)
	allServicesRe = phrases(`tutti i servizi`, `su tutti`, `tutto il sistema`, `intero sistema`, `everything`, `in generale`, `in general`, `generale`, `for all`, `across all`, `whole system`, `entire system`, `all services`, `every service`, `each service`, `ogni servizio`, `overall`, `globally`, `globale`, `ovunque`, `everywhere`, `togli (il )?filtro`, `senza filtro`, `remove (the )?filter`, `no filter`, `without (the )?filter`)
	resetFocusRe  = phrases(`in generale`, `overall`, `generale`, `panoramica`, `overview`, `situazione`, `non solo`, `not just`, `not only`, `quadro completo`, `full picture`, `tutto quanto`)
	rerunRe       = phrases(`^(e |and |ok e |ok and )?(ora|adesso|now)$`, `rifai`, `rifallo`, `ripeti`, `di nuovo`, `again`, `refresh`, `aggiorna`, `update`, `ricontrolla`, `recheck`, `check again`, `(is it|e) (any )?better`, `va meglio`, `migliorat\w*`, `still`, `ancora`, `di nuovo`)
	reuseRe       = phrases(`same question`, `stessa domanda`, `stessa cosa`, `same thing`, `same for`, `idem`, `ripeti per`, `repeat for`, `do it again`, `for the same`, `come prima`, `like before`)
	everythingRe  = phrases(`tutti`, `tutte`, `tutto`, `all`, `everything`, `completo`, `completa`, `full`)
)

// isFollowUp reports whether the message continues the previous request
// instead of standing on its own.
func isFollowUp(plain string, scope Scope, in intent, focus string) bool {
	if in != intentDiagnose {
		return false
	}
	// A message that states both service and window is self-contained.
	if scope.Mentions > 0 && scope.Explicit {
		return false
	}
	if followUpStartRe.MatchString(plain) || followUpEndRe.MatchString(plain) {
		return true
	}
	// A diagnosis request without its own service or window continues the
	// conversation ("ci sono errori?", "e la latenza?").
	return true
}

// mergeScope applies what the new message states on top of the previous scope.
func mergeScope(prev, cur Scope, plain string) Scope {
	out := prev
	if cur.Explicit {
		out.From, out.To, out.Today = cur.From, cur.To, cur.Today
	}
	switch {
	case cur.Service != "":
		out.Service = cur.Service
	case allServicesRe.MatchString(plain):
		out.Service = ""
	}
	out.TraceID = ""
	out.Mentions = cur.Mentions
	return out
}

func resetsFocus(plain string) bool { return resetFocusRe.MatchString(plain) }

// ----------------------------------------------------------------- focus

var (
	latencyStems = []string{"latenz", "latenc", "lent", "rilent", "slow", "rallent", "ritard", "prestazion", "performance", "perf", "veloc", "speed"}
	// "errors instead of latency": the topic after these words is the one dropped.
	insteadOfRe  = regexp.MustCompile(`\b(instead of|invece di|invece che|al posto di|al posto della|rather than|piuttosto che) (the |la |il |gli |i |le )?\S+`)
	errorStems   = []string{"error", "errori", "errore", "err", "fail", "fallit", "fallis", "crash", "eccezion", "exception", "5xx", "4xx", "rott", "broke", "riavvi", "restart", "reboot"}
	trafficStems = []string{"traffic", "richiest", "request", "throughput", "rps", "carico", "load", "volume", "calo", "drop", "qps"}
	logStems     = []string{"log", "messagg", "message"}
	responseRe   = phrases(`tempi di risposta`, `tempo di risposta`, `response times?`, `quanto (ci )?mette`, `how long`)
)

// detectFocus returns the single topic the message is about, or "" when it is
// general or mixes topics.
func detectFocus(norm string) string {
	norm = insteadOfRe.ReplaceAllString(norm, " ")
	found := map[string]bool{}
	for _, tok := range tokens(norm) {
		switch {
		case percentileRe.MatchString(tok) || hasAnyPrefix(tok, latencyStems) || fuzzyIn(tok, []string{"latenza", "latency"}):
			found[focusLatency] = true
		case statusCodeRe.MatchString(tok) || hasAnyPrefix(tok, errorStems) || fuzzyIn(tok, []string{"errori", "errore", "errors"}):
			found[focusErrors] = true
		case hasAnyPrefix(tok, trafficStems):
			found[focusTraffic] = true
		case hasAnyPrefix(tok, logStems) && !strings.HasPrefix(tok, "login"):
			found[focusLogs] = true
		}
	}
	if responseRe.MatchString(norm) {
		found[focusLatency] = true
	}
	// "log di errore" is about logs, not a mix.
	if found[focusLogs] && found[focusErrors] && len(found) == 2 {
		return focusLogs
	}
	if len(found) != 1 {
		return focusGeneral
	}
	for f := range found {
		return f
	}
	return focusGeneral
}

func focusOfKind(kind string) string {
	switch kind {
	case KindLatency:
		return focusLatency
	case KindTrafficDrop, KindNoTraffic:
		return focusTraffic
	case KindLogPattern:
		return focusLogs
	}
	return focusErrors
}

// ------------------------------------------------------------ unsupported

const (
	unsupportedChart    = "chart"
	unsupportedAction   = "action"
	unsupportedAlert    = "alert"
	unsupportedExport   = "export"
	unsupportedBusiness = "business"
)

var (
	chartRe        = phrases(`grafic\w*`, `chart\w*`, `graph\w*`, `plot\w*`, `diagramm\w*`, `istogramm\w*`, `histogram\w*`, `(crea|creami|create|fammi|make|build|nuova|new) .{0,15}dashboard`)
	chartRequestRe = phrases(`fammi`, `fai`, `fare`, `disegna\w*`, `mostrami un`, `draw`, `plot`, `create`, `crea`, `creami`, `make`, `generate`, `genera`, `voglio un`, `sai fare`, `can you (make|draw|plot|show|create)`, `build`)
	// Where a chart is only the source of an observation: "nel grafico ho visto".
	chartSeenRe = phrases(`(nel|sul|dal|nella|sulla|in the|on the|from the) (grafico|chart|graph|dashboard)`, `ho visto`, `i saw`, `shows`, `has spikes`)
	opsWords    = map[string]bool{"riavvia": true, "riavviare": true, "riavviami": true, "restart": true, "reboot": true,
		"cancella": true, "cancellare": true, "elimina": true, "eliminare": true, "delete": true, "rimuovi": true,
		"rimuovere": true, "remove": true, "kill": true, "spegni": true, "spegnere": true, "shutdown": true,
		"purge": true, "svuota": true, "svuotare": true, "pulisci": true, "pulire": true, "flush": true,
		"truncate": true, "revert": true, "deploya": true, "scala": true, "scalare": true, "ferma": true, "fermare": true, "stop": true}
	opsFuzzy    = []string{"restart", "rollback", "riavvia", "delete", "cancella", "elimina"}
	opsPhraseRe = phrases(`(fai|fare|fammi|do|esegui|lancia|run|puoi fare|can you do|start) (un |il |a |the )?(rollback|rollbak|roll back|deploy|restart|riavvio|redeploy)`,
		`torna(re)? alla versione`, `go back to the (previous )?version`, `clear (the )?cache`, `aumenta le repliche`, `diminuisci le repliche`,
		`scale (up|down|out|in)`, `roll back`, `rollback (di|del|of)`)
	alertCfgRe = regexp.MustCompile(`\b(crea|creami|creare|imposta|impostami|impostare|configura|configurami|configurare|settami|set up|setup|set|create|add|aggiungi|attiva|metti|mettimi|mettere)\b.{0,40}\b(alert\w*|allarm\w*|notific\w*|regol\w*|rules?|sogli\w*|threshold\w*|avvis\w*)\b|` +
		`\b(mandami|send me|avvisami|notify me|alert me|avvertimi|scrivimi|email me|ping me)\b.{0,40}\b(se|quando|if|when|appena|as soon as)\b`)
	exportRe   = phrases(`esporta\w*`, `export\w*`, `scarica\w*`, `download\w*`, `csv`, `pdf`, `excel`, `xlsx`)
	businessRe = phrases(`quanti utenti`, `how many users`, `utenti attivi`, `active users`, `fatturato`, `revenue`, `vendite`, `sales`, `conversion\w*`, `conversioni`, `quanti ordini`, `how many orders`, `numero di ordini`, `ordini (fatti|di oggi|ricevuti)`, `incassi`)
	infraRe    = phrases(`cpu`, `memoria`, `memory`, `ram`, `disco`, `disk`, `heap`, `oom`, `memory leak`)
	// Past or descriptive forms: "si e riavviato", "i pod si riavviano", "dopo il rollback".
	opsEventRe = phrases(`riavviat\w*`, `si riavvi\w*`, `restarted`, `restarting`, `restarts`, `rebooted`, `crashed`, `deleted`, `cancellat\w*`, `eliminat\w*`, `(dopo|prima|after|before|since) (il |lo |the |a |un )?(rollback|deploy|restart|riavvio)`, `rolled back`, `si ferma`, `it stops`, `keeps stopping`)
)

// unsupportedReason names what the message asks for that the assistant cannot
// do. Questions about why something happened are diagnoses, not requests.
func unsupportedReason(plain string) string {
	cause := causeRe.MatchString(plain)
	switch {
	case chartRe.MatchString(plain) && !chartSeenRe.MatchString(plain) && !cause:
		return unsupportedChart
	case alertCfgRe.MatchString(plain):
		return unsupportedAlert
	case !cause && !opsEventRe.MatchString(plain) && isOpsRequest(plain):
		return unsupportedAction
	case exportRe.MatchString(plain):
		return unsupportedExport
	case businessRe.MatchString(plain):
		return unsupportedBusiness
	}
	return ""
}

func isOpsRequest(plain string) bool {
	if opsPhraseRe.MatchString(plain) {
		return true
	}
	toks := tokens(plain)
	for i, tok := range toks {
		if i > 0 && (toks[i-1] == "si" || toks[i-1] == "it") {
			continue
		}
		if opsWords[tok] {
			return true
		}
		if len(tok) >= 6 {
			for _, w := range opsFuzzy {
				if withinEdits(tok, w, 1) {
					return true
				}
			}
		}
	}
	return false
}

// fuzzyDetail catches typos of the details keywords ("dettgli", "detials").
func fuzzyDetail(plain string) bool {
	for _, tok := range tokens(plain) {
		if len(tok) >= 6 && (withinEdits(tok, "dettagli", 2) || withinEdits(tok, "details", 1)) && !strings.HasPrefix(tok, "dettagliat") {
			return true
		}
	}
	return false
}
