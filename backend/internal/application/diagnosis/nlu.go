package diagnosis

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// This file turns a free-text chat message (Italian or English, often sloppy)
// into what the diagnosis needs: an intent, a time window, a service, a trace
// id and the language to answer in. It is deliberately rule based: the target
// deployment has no room for a language model. testdata/nlu_corpus.json holds
// the labeled messages these rules are checked against.

type intent int

const (
	intentDiagnose intent = iota
	intentHelp
	intentGreeting
	intentThanks
	intentUnclear
)

// ---------------------------------------------------------- normalization

var (
	accentReplacer = strings.NewReplacer(
		"à", "a", "á", "a", "â", "a", "è", "e", "é", "e", "ê", "e", "ì", "i", "í", "i",
		"ò", "o", "ó", "o", "ô", "o", "ù", "u", "ú", "u", "’", "'", "‘", "'",
	)
	// Emoji people use instead of words.
	emojiReplacer = strings.NewReplacer(
		"👋", " ciao ", "👍", " ok ", "🙏", " grazie ", "🙌", " grazie ", "👌", " ok ",
		"💪", " grande ", "🔥", " fire ", "🚨", " alert ", "💥", " crash ", "🐢", " slow ",
	)
	nonWordRe = regexp.MustCompile(`[^a-z0-9\-.,:/?#]+`)
	spacesRe  = regexp.MustCompile(`\s+`)
	tokenRe   = regexp.MustCompile(`[a-z0-9]+(?:[-.,][a-z0-9]+)*`)
)

// normalize lowercases, maps emoji to words, strips accents, collapses
// stretched letters ("ciaoooo") and turns apostrophes and punctuation into
// spaces so that "Perché l'ultima mezz'ora?" reads "perche l ultima mezz ora?".
func normalize(s string) string {
	s = emojiReplacer.Replace(strings.ToLower(s))
	s = accentReplacer.Replace(s)
	s = collapseRepeats(s)
	s = nonWordRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(spacesRe.ReplaceAllString(s, " "))
}

// collapseRepeats squeezes runs of 3+ identical letters to one: "heyyy" -> "hey".
func collapseRepeats(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		j := i
		for j+1 < len(runes) && runes[j+1] == runes[i] {
			j++
		}
		if j-i >= 2 && runes[i] >= 'a' && runes[i] <= 'z' {
			b.WriteRune(runes[i])
		} else {
			b.WriteString(string(runes[i : j+1]))
		}
		i = j
	}
	return b.String()
}

func tokens(norm string) []string { return tokenRe.FindAllString(norm, -1) }

// ---------------------------------------------------------------- trace id

// A trace id is exactly 32 hex chars not glued to other hex chars: that rules
// out longer hashes while accepting ids inside URLs, backticks or punctuation.
var traceIDRe = regexp.MustCompile(`(?:^|[^0-9a-f])([0-9a-f]{32})(?:[^0-9a-f]|$)`)

func findTraceID(prompt string) string {
	if m := traceIDRe.FindStringSubmatch(strings.ToLower(prompt)); m != nil {
		return m[1]
	}
	return ""
}

// ------------------------------------------------------------------ window

var numberWords = map[string]float64{
	"un": 1, "uno": 1, "una": 1, "a": 1, "an": 1, "one": 1,
	"due": 2, "two": 2, "paio": 2, "couple": 2, "tre": 3, "three": 3, "qualche": 3, "few": 3,
	"quattro": 4, "four": 4, "cinque": 5, "five": 5, "sei": 6, "six": 6, "sette": 7, "seven": 7,
	"otto": 8, "eight": 8, "nove": 9, "nine": 9, "dieci": 10, "ten": 10, "undici": 11, "eleven": 11,
	"dodici": 12, "twelve": 12, "quindici": 15, "fifteen": 15, "venti": 20, "twenty": 20,
	"ventiquattro": 24, "trenta": 30, "thirty": 30, "quaranta": 40, "forty": 40,
	"quarantacinque": 45, "quarantotto": 48, "cinquanta": 50, "fifty": 50, "sessanta": 60,
	"sixty": 60, "novanta": 90, "ninety": 90,
}

const (
	unitMinute = time.Minute
	unitHour   = time.Hour
	unitDay    = 24 * time.Hour
	unitWeek   = 7 * unitDay
	unitMonth  = 30 * unitDay
)

func unitOf(word string) (time.Duration, bool) {
	switch word {
	case "m", "min", "mins", "minuto", "minuti", "minute", "minutes":
		return unitMinute, true
	case "h", "hr", "hrs", "ora", "ore", "hour", "hours":
		return unitHour, true
	case "g", "gg", "d", "giorno", "giorni", "giornata", "giornate", "day", "days":
		return unitDay, true
	case "w", "wk", "wks", "settimana", "settimane", "week", "weeks":
		return unitWeek, true
	case "mese", "mesi", "month", "months":
		return unitMonth, true
	}
	return 0, false
}

// Short units only count after digits ("15m", "2 h"): "sei m" or "a d" are
// not durations.
var shortUnits = map[string]bool{"m": true, "h": true, "g": true, "gg": true, "d": true, "w": true,
	"hr": true, "hrs": true, "min": true, "mins": true, "wk": true, "wks": true}

// A duration right after these words is not a lookback: "tra un ora", "in 5
// minutes", "ho 2 minuti", "i have 5 min".
var notLookback = map[string]bool{"tra": true, "fra": true, "in": true, "entro": true, "ho": true,
	"have": true, "got": true, "hai": true, "abbiamo": true, "dopo": true, "after": true}

// These words before a duration mark it as the window the user asks about.
var lookbackWords = map[string]bool{"ultimi": true, "ultime": true, "ultima": true, "ultimo": true,
	"last": true, "past": true, "previous": true, "scorsi": true, "scorse": true, "da": true,
	"dalle": true, "since": true, "over": true, "negli": true, "nelle": true, "nell": true,
	"nella": true, "nel": true}

var (
	gluedDurationRe = regexp.MustCompile(`^(\d+(?:[.,]\d+)?)([a-z]+)$`)
	compoundHourRe  = regexp.MustCompile(`^(\d+)h(\d+)m?$`)
	hourAndHalfRe   = regexp.MustCompile(`\b(\d+|un|una|an|one|due|two)\s*(?:ora|ore|h|hour|hours)\s+(?:e|and)\s+(?:mezza|mezzo|a half)\b`)
	lastUnitRe      = regexp.MustCompile(`\b(ultim[aoei]|last|past|previous|scors[aoei]|questo|questa|this)\s+(ora|ore|hour|hours|hr|hrs|minuti|minutes|mins|giorno|giorni|giornata|day|days|settimana|settimane|week|weeks|mese|month)\b`)
	halfHourRe      = regexp.MustCompile(`\b(mezz ora|mezzora|mezza ora|mezz oretta|half an hour|half hour|half-hour)\b`)
	quarterRe       = regexp.MustCompile(`\b(quarto d ora|quarter of an hour|quarter hour|quarter-hour)\b`)
	threeQrtRe      = regexp.MustCompile(`\b(tre quarti d ora|three quarters of an hour)\b`)
	todayRe         = regexp.MustCompile(`\b(oggi|today|stamattina|stamani|stamane|stasera|stanotte|this morning|this afternoon|this evening|tonight|in giornata|since midnight|da mezzanotte|dalla mezzanotte)\b`)
	dayBeforeRe     = regexp.MustCompile(`\b(avantieri|l altro ieri|day before yesterday)\b`)
	yesterdayRe     = regexp.MustCompile(`\b(ieri|yesterday|last night|ieri sera|ieri notte)\b`)
	weekRe          = regexp.MustCompile(`\b(settimana|week|weekend)\b`)
	monthRe         = regexp.MustCompile(`\b(mese|month)\b`)
	futureDayRe     = regexp.MustCompile(`\b(domani|tomorrow|prossima settimana|next week|lunedi|martedi|mercoledi|giovedi|venerdi)\b`)
)

type durationHit struct {
	value    time.Duration
	lookback bool
}

// durationsIn finds "15m", "2 ore", "1.5h", "1h30m", "due ore", "a couple of
// hours", "un paio d ore", skipping future ones ("tra un ora").
func durationsIn(toks []string) (hits []durationHit, mentioned bool) {
	for i := 0; i < len(toks); i++ {
		prev := ""
		if i > 0 {
			prev = toks[i-1]
		}
		prev2 := ""
		if i > 1 {
			prev2 = toks[i-2]
		}
		var value time.Duration
		found := false
		tok := toks[i]
		if m := compoundHourRe.FindStringSubmatch(tok); m != nil {
			h, _ := strconv.Atoi(m[1])
			mins, _ := strconv.Atoi(m[2])
			value, found = time.Duration(h)*time.Hour+time.Duration(mins)*time.Minute, true
		} else if m := gluedDurationRe.FindStringSubmatch(tok); m != nil {
			if unit, ok := unitOf(m[2]); ok {
				value, found = time.Duration(parseNumber(m[1])*float64(unit)), true
			}
		} else if n, isNum := numberToken(tok); isNum && i+1 < len(toks) {
			j := i + 1
			if (toks[j] == "di" || toks[j] == "d" || toks[j] == "of") && j+1 < len(toks) {
				j++
			}
			if unit, ok := unitOf(toks[j]); ok && (isDigits(tok) || !shortUnits[toks[j]]) {
				value, found = time.Duration(n*float64(unit)), true
				// "un paio d ore": the number word "paio" follows "un".
				if tok == "un" || tok == "a" {
					if _, ok := numberWords[toks[i+1]]; ok {
						found = false
					}
				}
			}
		}
		if !found {
			continue
		}
		if notLookback[prev] || (numberWords[prev] == 1 && notLookback[prev2]) {
			continue
		}
		mentioned = true
		if value <= 0 {
			continue
		}
		hits = append(hits, durationHit{value: value, lookback: lookbackWords[prev] || lookbackWords[prev2]})
	}
	return hits, mentioned
}

func numberToken(tok string) (float64, bool) {
	if n, ok := numberWords[tok]; ok {
		return n, true
	}
	if isDigits(tok) || strings.ContainsAny(tok, ".,") && isDigits(strings.NewReplacer(".", "", ",", "").Replace(tok)) {
		return parseNumber(tok), true
	}
	return 0, false
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseNumber(s string) float64 {
	if n, ok := numberWords[s]; ok {
		return n
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0
	}
	return n
}

// parseWindow returns the lookback the message asks for (capped to
// maxWindow), whether one was stated, and whether a window was mentioned at
// all (true for "ultimi 0 minuti", which still asks for a diagnosis).
func parseWindow(norm string, now time.Time) (window time.Duration, explicit, mentioned bool) {
	toks := tokens(norm)
	hits, mentioned := durationsIn(toks)
	switch {
	case hourAndHalfRe.MatchString(norm) && !futureWindow(norm, hourAndHalfRe):
		m := hourAndHalfRe.FindStringSubmatch(norm)
		window = time.Duration(parseNumber(m[1])*60+30) * time.Minute
	case threeQrtRe.MatchString(norm) && !futureWindow(norm, threeQrtRe):
		window = 45 * time.Minute
	case halfHourRe.MatchString(norm) && !futureWindow(norm, halfHourRe):
		window = 30 * time.Minute
	case quarterRe.MatchString(norm) && !futureWindow(norm, quarterRe):
		window = 15 * time.Minute
	case len(hits) > 0:
		window = hits[0].value
		for _, h := range hits {
			if h.lookback {
				window = h.value
				break
			}
		}
	}
	if window == 0 {
		if m := lastUnitRe.FindStringSubmatch(norm); m != nil {
			window = lastUnitWindow(m[2])
		}
	}
	if window == 0 && !futureDayRe.MatchString(norm) {
		switch {
		case todayRe.MatchString(norm):
			window = sinceMidnight(now)
		case dayBeforeRe.MatchString(norm):
			window = 2 * unitDay
		case yesterdayRe.MatchString(norm):
			window = unitDay
		case monthRe.MatchString(norm):
			window = unitMonth
		case weekRe.MatchString(norm) && !strings.Contains(norm, "weekend"):
			window = unitWeek
		}
	}
	if window <= 0 {
		return 0, false, mentioned
	}
	if window < time.Minute {
		window = time.Minute
	}
	if window > maxWindow {
		window = maxWindow
	}
	return window, true, true
}

func sinceMidnight(now time.Time) time.Duration {
	y, mo, d := now.Date()
	return now.Sub(time.Date(y, mo, d, 0, 0, 0, 0, now.Location()))
}

// futureWindow reports whether the phrase re matched follows "tra"/"in"/"fra"
// ("ci risentiamo tra mezz ora").
func futureWindow(norm string, re *regexp.Regexp) bool {
	loc := re.FindStringIndex(norm)
	if loc == nil {
		return false
	}
	before := tokens(norm[:loc[0]])
	return len(before) > 0 && notLookback[before[len(before)-1]]
}

// lastUnitWindow maps "last hour" to one unit and "last hours" (plural, no
// number) to a few of them.
func lastUnitWindow(unit string) time.Duration {
	switch unit {
	case "ora", "hour", "hr", "giorno", "giornata", "day", "settimana", "week", "mese", "month":
		u, _ := unitOf(unit)
		return u
	case "minuti", "minutes", "mins":
		return 15 * time.Minute
	case "ore", "hours", "hrs":
		return 3 * time.Hour
	case "giorni", "days":
		return 3 * unitDay
	case "settimane", "weeks":
		return unitWeek
	}
	return 0
}

// ----------------------------------------------------------------- service

// genericNameParts are dropped when deriving aliases from service names:
// "payment-service" answers to "payment", "api-gateway" to "gateway".
var genericNameParts = map[string]bool{
	"service": true, "services": true, "svc": true, "api": true, "server": true, "srv": true,
	"app": true, "backend": true, "worker": true, "prod": true, "v1": true, "v2": true,
}

// italianAliases maps the distinctive English word of a service name to the
// Italian words people use for the same domain.
var italianAliases = map[string][]string{
	"payment":      {"pagamento", "pagamenti"},
	"payments":     {"pagamento", "pagamenti"},
	"cart":         {"carrello", "carrelli"},
	"shipping":     {"spedizione", "spedizioni"},
	"catalog":      {"catalogo", "cataloghi"},
	"inventory":    {"inventario", "magazzino"},
	"notification": {"notifica", "notifiche"},
	"auth":         {"autenticazione", "login"},
	"order":        {"ordine", "ordini"},
	"search":       {"ricerca"},
	"user":         {"utente", "utenti"},
}

type serviceAliases struct {
	exact map[string]string // alias -> service name
	fuzzy map[string]string // aliases long enough for typo matching
}

func buildAliases(services []string) serviceAliases {
	exact := map[string]string{}
	fuzzy := map[string]string{}
	shared := map[string]bool{}
	add := func(alias, service string) {
		if prev, ok := exact[alias]; ok && prev != service {
			shared[alias] = true
			return
		}
		exact[alias] = service
		if len(alias) >= 6 && !strings.Contains(alias, " ") {
			fuzzy[alias] = service
		}
	}
	for _, service := range services {
		name := strings.ToLower(strings.TrimSpace(service))
		if name == "" {
			continue
		}
		add(name, service)
		parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
		distinctive := []string{}
		for _, p := range parts {
			if !genericNameParts[p] && len(p) >= 3 {
				distinctive = append(distinctive, p)
			}
		}
		if len(distinctive) > 0 {
			joined := strings.Join(distinctive, "-")
			add(joined, service)
			add(strings.Join(distinctive, " "), service)
			add(joined+"s", service)
			for _, it := range italianAliases[joined] {
				add(it, service)
			}
		}
		add(strings.Join(parts, " "), service) // "api gateway"
	}
	for alias := range shared {
		delete(exact, alias)
		delete(fuzzy, alias)
	}
	return serviceAliases{exact: exact, fuzzy: fuzzy}
}

// findServices returns every service the message refers to.
func findServices(norm string, services []string) []string {
	if len(services) == 0 {
		return nil
	}
	aliases := buildAliases(services)
	toks := tokens(norm)
	seen := map[string]bool{}
	out := []string{}
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if i+1 < len(toks) {
			if s, ok := aliases.exact[tok+" "+toks[i+1]]; ok {
				add(s)
				i++
				continue
			}
		}
		if s, ok := aliases.exact[tok]; ok {
			add(s)
			continue
		}
		if len(tok) >= 5 && !commonWords[tok] {
			for alias, s := range aliases.fuzzy {
				if withinEdits(tok, alias, 1) {
					add(s)
					break
				}
			}
		}
	}
	return out
}

// ------------------------------------------------------------------ intent

func phrases(alternatives ...string) *regexp.Regexp {
	return regexp.MustCompile(`\b(` + strings.Join(alternatives, "|") + `)\b`)
}

var (
	// Questions about the assistant itself.
	helpPhraseRe = phrases(
		`(che )?cos?a? (puoi|sai|riesci a|potresti) fa(re)?`, `che sai fa(re)?`, `cosa fai`, `che fai`,
		`cos altro (sai|puoi)`, `che altro (sai|puoi)`, `cosa non (sai|puoi) fare`, `di cosa sei capace`,
		`come funzion\w*$`, `come funzion\w* (questa|sta|la|questo) (chat|cosa|roba|app|tool|bot|assistente|coso)`,
		`come funzioni`, `come ti (uso|si usa|posso usare|devo usare|devo fare|devo chiedere|chiedo|parlo)`,
		`come si usa`, `come posso usarti`, `come faccio a (usarti|chiedert\w*)`, `mi spieghi come`,
		`a cosa (servi|serve (questa|sta|questo|il bot|l assistente))`, `chi sei`, `(che )?cosa sei`,
		`sei (un |una )?(bot|ia|ai|intelligenza artificiale|umano|persona|robot|chatgpt|intelligente)`,
		`(cosa|che domande) (posso|devo) (chiederti|farti|domandarti)`, `che domande (posso|ti posso) fare`,
		`posso (chiederti|incollarti|darti|mandarti|scriverti|farti|usarti|chiedere)`,
		`(dammi|fammi|mostrami) (degli |qualche |un )?esemp\w*`, `esempi di domande`,
		`(che|quali) servizi (conosci|vedi|monitori|hai|segui)`, `(which|what) services (do|can) you`,
		`fino a quanto indietro`, `quanto indietro`, `how far back`, `da dove prendi`, `dove prendi i dati`,
		`che dati (usi|guardi|leggi)`, `what data do you`, `where do you get`,
		`(parli|capisci|funzioni anche in|funziona anche in) (italiano|inglese)`,
		`(speak|understand|support) (italian|english)`, `parole chiave`, `keywords`,
		`se ti scrivo`, `quando te lo chiedo`, `devo scrivere`, `basta (scrivere|tipo)`, `che differenza`,
		`cosa (significa|vuol dire)`, `what does .+ mean`, `what s the difference`, `whats the difference`,
		`what is the difference`, `(explain|spiegami|spieghi) (the |your |la |l |tua )?(last |ultima )?(answer|risposta|report)`,
		`da dove (inizio|parto|comincio)`, `sono nuovo`, `where do i start`, `i m new`, `getting started`,
		`what (can|do|could) (you|u) do`, `what else can you`, `what can t you do`, `what are you`, `who are you`,
		`what is this( thing| tool| bot| assistant)?`, `what s this( thing| tool| bot| assistant)?( for)?`,
		`whats this( thing| tool| bot| assistant)?`, `(how|hw) (do i|can i|should i|to) (use|talk to|ask|search|find|look up|query|filter|paste|write|type)`,
		`(how|hw) (does|do) (it|this|you|this thing) work`, `is this (a bot|a real person|an ai|a human)`,
		`real person`, `are you (a bot|an ai|a human|human|a person|a robot|chatgpt|an llm)`,
		`what (can|should) i ask`, `(give|show) me (some |an )?examples?`, `what kind of`, `che tipo di`,
		`what are your (capabilities|features|commands|limits)`, `do you support`, `supporti`,
	)
	// "sai / can you / riesci a ..." ask about ability rather than for an action.
	abilityRe = phrases(
		`sai`, `riesci( a)?`, `sei in grado di`, `sei capace di`, `e possibile`, `si puo`, `puoi`, `potresti`,
		`hai accesso`, `leggi anche`, `vedi anche`, `can (you|u|this|it)`, `could (you|u)`, `do you`,
		`does (it|this)`, `are you able to`, `is it possible to`, `have access`,
	)
	// Status and incident questions: "tutto ok?", "is prod healthy?".
	statusPhraseRe = phrases(
		`tutto (ok|okay|bene|a posto|regolare|tranquillo|rosso|giu|fermo)`, `va tutto`, `funziona tutto`, `tutto funziona`,
		`non (va|vanno|funziona|funzionano|risponde|rispondono|carica|caricano|parte|partono|gira|girano)`,
		`cosa (non va|succede|sta succedendo|e successo|c e che non va|si e rotto)`, `che (succede|sta succedendo|e successo)`,
		`ci sono (stati )?(errori|problemi|anomalie|issue|alert|disservizi|incidenti)`, `(ce|c e) (qualche |un )?(errore|problema|anomalia|disservizio)`,
		`(abbiamo|avete|hanno) (problemi|errori)`, `e (tutto )?rosso`, `dashboard rossa`, `tutto rosso`,
		`(everything|all|it) (ok|okay|good|fine|alright|working)`, `all good`, `(are there|is there|any) (any )?(issues|problems|errors|alerts|incidents)`,
		`(anything|something|qualcosa) (is |e )?(wrong|broken|failing|down|weird|odd|off|on fire|rotto|giu|che non va)`,
		`what s (wrong|broken|going on|up with|happening)`, `tutt[ao] ross[ao]`,
		`whats (wrong|broken|going on|up with|happening)`, `what (happened|is going on|is happening|broke)`,
		`not working`, `doesn t work`, `isn t working`, `stopped working`, `or everything`, `o tutto`,
		`is (it|prod|production|the system|everything) (up|down|ok|healthy|working|alive)`,
		`come (sta|stanno|va|vanno|procede|procedono|sta andando|butta)\b.{0,30}\b(sistema|servizi|servizio|prod|produzione|infra|piattaforma|app|cose|situazione)`,
		`com e la situazione`, `come siamo messi`, `situazione`,
		`how (is|are|s) (prod|production|the system|things|everything|the services)`,
	)
	// Imperative requests that turn a help-looking message with a target into an action.
	requestRe = phrases(
		`controlla\w*`, `check`, `verifica`, `analizza`, `guarda`, `guardami`, `mostrami`, `mostra`, `show me`,
		`dimmi`, `tell me`, `fammi vedere`, `mi fai vedere`, `let me see`, `take a look`, `cerca`, `find`,
		`look (at|into)`, `have a look`, `investigate`, `analy[sz]e`, `dammi`, `give me`, `look up`,
		`(dai|darci|dare|dagli|dargli|butta|buttare|da) un occhi\w*`, `can (you|u) (check|look)`, `fai (una|la) diagnosi`, `run a`,
	)
	// Verbs that mean acting on a target in ability questions ("puoi controllare payment?").
	actionStems = []string{"controll", "check", "verific", "analizz", "analy", "guard", "look",
		"vedere", "vedi", "see", "mostr", "show", "dimm", "tell", "cerc", "find", "trov", "investig",
		"esamin", "examin", "legg", "read", "monitor", "dirmi", "occhi"}
	// Messages that are about the bot misbehaving, not about the system.
	botFrustrationRe = phrases(
		`(il |this |the |sto |questo )?(bot|chat|assistente|assistant|coso) (non (funziona|funzioni|capisce|capisci|serve)|doesn t work|is broken|is useless|useless|inutile|stupid\w*)`,
		`(you|u) (are|re|r) (useless|stupid|dumb|broken|wrong)`, `sei (inutile|stupido|stupida|scemo|scema)`,
		`non capisci (niente|nulla|un cazzo)`, `useless bot`, `non servi a niente`, `this thing (doesn t work|is useless)`,
	)
	// Off-topic requests: creative ones always win, everyday ones only when no
	// real problem is reported.
	creativeRe = phrases(`poesia`, `poema`, `poem`, `haiku`, `barzellett\w*`, `joke`, `jokes`, `battuta`, `canzone`, `song`, `racconto`, `story`, `limerick`)
	offTopicRe = phrases(
		`meteo`, `weather`, `che tempo fa`, `sushi`, `pizza`, `pranzo`, `lunch`, `cena`, `dinner`, `ristorante`,
		`restaurant`, `calcio`, `football`, `soccer`, `partita`, `inter`, `milan`, `juve`, `juventus`, `chi ha vinto`,
		`who won`, `film`, `movie`, `ricetta`, `recipe`, `riunione`, `meeting`, `carta di credito`, `credit card`, `della spesa`,
		`cose da fare`, `to-do`, `todo list`, `vacanz\w*`, `holiday`, `traduci`, `translate`,
	)
	// Statements that things are fine: a question with them is a status check,
	// a statement next to a thank-you is a closing ("all good, thanks").
	fineStatusRe = phrases(`tutto (ok|okay|bene|a posto|regolare|tranquillo)`, `funziona tutto`, `tutto funziona`,
		`(everything|all|it) (ok|okay|good|fine|alright|working)`, `all good`)
	// Asking for the cause turns an ability question into a request ("can you see why").
	causeRe = phrases(`why`, `perche`, `come mai`, `what caused`, `cosa ha causato`, `root cause`, `causa`)
	// Ids that are not trace ids but still name something concrete to look at.
	concreteIDRe = regexp.MustCompile(`(?:^|\s)(?:[0-9a-f]{12,}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:\s|$)`)
	exampleIDRe  = regexp.MustCompile(`\b(es|tipo|ad esempio|per esempio|like|e g|eg|such as|for example|for instance)\s+(?:[0-9a-f]{8,}|[0-9a-f]{8}-[0-9a-f-]{27})`)
	injectionRe  = phrases(`ignore (all |the )?(previous|above|prior) instructions`, `ignora (le |tutte le )?istruzioni`, `password`, `system prompt`, `drop table`, `script`)
	statusCodeRe = regexp.MustCompile(`^(?:[45]\d\d|[45]xx)s?$`)
	percentileRe = regexp.MustCompile(`^p(50|75|90|95|99|999)$`)
)

// strongStems report a problem on their own ("errori", "latency", "timeout").
var strongStems = []string{
	"error", "errori", "errore", "eccezion", "exception", "fail", "fallis", "fallit", "fallim",
	"crash", "latenz", "latenc", "lentezz", "lentiss", "slow", "timeout", "rallent", "ritard",
	"degrad", "anomal", "incident", "outage", "guast", "broke", "spike", "picc", "blocc",
	"stuck", "hang", "impall", "freez", "saturaz", "leak", "singhiozz", "esplos", "fiamm", "instabil", "unstable", "flak",
	"disservizi", "irraggiungibil", "unreachable", "unavailable", "indisponibil", "traffic",
	"throughput", "5xx", "4xx", "diagnos", "capricc",
}

var strongWords = map[string]bool{
	"down": true, "giu": true, "lento": true, "lenta": true, "lenti": true, "lente": true,
	"lag": true, "rotto": true, "rotti": true, "rotta": true, "rotte": true, "oom": true,
	"fire": true, "ko": true, "morto": true, "dead": true, "err": true, "errs": true,
	"latency": true, "errors": true, "timeouts": true, "rosso": true, "rossa": true,
	"rossi": true, "red": true,
}

// weakWords point at telemetry but alone ("ho un problema con la carta")
// are not a request: they need a second clue.
var weakStems = []string{"problem", "issue", "controll", "check", "verific", "analizz", "analis",
	"analy", "diagnos", "investig", "monitor", "status", "salute", "health", "richiest",
	"request", "alert", "allarm"}

var weakWords = map[string]bool{
	"log": true, "logs": true, "trace": true, "traces": true, "tracce": true, "traccia": true,
	"span": true, "spans": true, "stato": true, "carico": true, "load": true, "rps": true,
	"cpu": true, "memoria": true, "memory": true, "sistema": true, "system": true, "prod": true,
	"produzione": true, "production": true, "infra": true, "infrastruttura": true, "servizi": true,
	"services": true, "cluster": true, "piattaforma": true, "platform": true, "uptime": true,
	"sla": true, "slo": true, "bug": true, "bugs": true, "dashboard": true, "metriche": true,
	"metrics": true, "deploy": true, "rilascio": true, "release": true, "backend": true,
	"frontend": true, "api": true, "server": true, "servers": true, "db": true, "database": true,
	"sito": true, "site": true, "website": true, "healthy": true, "unhealthy": true,
}

// typoTargets are canonical problem words for typo matching ("erori", "latenzza").
var typoTargets = []string{
	"errori", "errore", "errors", "error", "latenza", "latency", "lento", "lenta", "timeout",
	"traffico", "traffic", "anomalie", "anomaly", "eccezioni", "exceptions", "crash", "failed",
	"failure", "fallito", "fallisce", "degradato", "rallentamento", "rallentato", "slowness",
}

var weakTypoTargets = []string{"problema", "problemi", "problem", "problems", "analizza", "analyze",
	"controlla", "verifica", "richieste", "requests"}

// commonWords never count as typos of a service or problem word.
var commonWords = map[string]bool{
	"about": true, "after": true, "again": true, "cosa": true, "could": true, "there": true,
	"these": true, "those": true, "where": true, "which": true, "would": true, "quale": true,
	"quali": true, "quando": true, "questo": true, "questa": true, "quello": true, "terror": true,
	"mirror": true, "clash": true, "crush": true, "cheek": true, "carta": true, "ciao": true,
	"grazie": true, "sempre": true, "ancora": true, "adesso": true, "lentils": true, "letter": true,
	"better": true, "errand": true, "errare": true, "latina": true, "latino": true,
}

var greetingWords = map[string]bool{
	"ciao": true, "salve": true, "buongiorno": true, "buonasera": true, "buonpomeriggio": true,
	"buondi": true, "hey": true, "ehi": true, "ei": true, "hei": true, "hi": true, "hello": true,
	"hallo": true, "hola": true, "yo": true, "sup": true, "howdy": true, "bella": true, "we": true,
	"ue": true, "hiya": true, "heya": true, "morning": true, "buon": true, "pomeriggio": true,
	"good": true, "afternoon": true, "evening": true, "giorno": true, "sera": true, "night": true,
	"buonanotte": true, "notte": true, "goodnight": true, "salut": true,
}

var greetingStems = []string{"buongiorn", "buonaser", "buonpomer", "ciao", "hell"}

var thanksWords = map[string]bool{
	"grazie": true, "grazi": true, "grz": true, "thanks": true, "thank": true, "thx": true, "ty": true,
	"tnx": true, "tks": true, "thnx": true, "ok": true, "okay": true, "okk": true, "okok": true,
	"okey": true, "oki": true, "kk": true, "k": true, "perfetto": true, "perfect": true, "ottimo": true,
	"great": true, "cool": true, "nice": true, "top": true, "capito": true, "understood": true,
	"chiaro": true, "benissimo": true, "fantastico": true, "awesome": true, "bye": true,
	"arrivederci": true, "cya": true, "later": true, "presto": true, "prossima": true,
	"alright": true, "bravo": true, "brava": true, "grande": true, "mitico": true, "super": true,
	"gentile": true, "gentilissimo": true, "cheers": true, "appreciate": true, "appreciated": true,
	"merci": true, "gracias": true, "risentiamo": true, "sentiamo": true, "vediamo": true,
	"ottima": true, "perfetta": true, "fatto": true, "done": true,
}

// positiveWords say that things are fine: with them a thank-you that names a
// service stays a thank-you ("grazie, spedizione ok adesso").
var positiveWords = map[string]bool{
	"ok": true, "okay": true, "fine": true, "bene": true, "posto": true, "risolto": true,
	"resolved": true, "fixed": true, "sistemato": true, "worked": true, "works": true,
	"good": true, "recovered": true, "ripreso": true, "tornato": true, "tornata": true,
	"back": true, "looked": true, "looks": true, "funziona": true, "funzionato": true,
}

// fillerWords may accompany a greeting without changing it.
var fillerWords = map[string]bool{
	"a": true, "all": true, "tutti": true, "there": true, "anyone": true, "anybody": true,
	"everyone": true, "team": true, "ragazzi": true, "raga": true, "you": true, "u": true,
	"2": true, "e": true, "and": true, "di": true, "nuovo": true, "again": true, "bot": true,
	"assistente": true, "assistant": true, "come": true, "va": true, "stai": true, "butta": true,
	"how": true, "are": true, "doing": true, "s": true, "it": true, "going": true, "looking": true,
	"tu": true, "te": true, "the": true, "is": true, "to": true, "whats": true, "what": true,
	"up": true, "man": true, "bro": true, "dude": true, "si": true, "yes": true, "yeah": true,
	"sei": true, "un": true, "una": true, "qui": true, "here": true, "c": true, "ce": true,
}

var helpWords = map[string]bool{
	"aiuto": true, "help": true, "halp": true, "aiutami": true, "aiutarmi": true, "aiuti": true,
	"istruzioni": true, "instructions": true, "esempi": true, "esempio": true, "examples": true,
	"example": true, "comandi": true, "commands": true, "guida": true, "guide": true,
	"manuale": true, "manual": true, "tutorial": true, "usage": true, "howto": true, "faq": true,
	"capabilities": true, "funzionalita": true, "features": true, "chatgpt": true, "llm": true,
	"gpt": true, "openai": true,
}

type signals struct {
	strong, status, request, ability, action, helpPhrase, help bool
	greeting, thanks, positive, creative, offTopic, injection  bool
	botFrustration, question, onlySocial                       bool
	fineStatus, cause, concreteID                              bool
	weak                                                       int
}

func readSignals(prompt, norm string) signals {
	// Phrase rules run on the text without question marks so that "$" and
	// word-boundary anchors behave; the question mark is its own signal.
	plain := strings.TrimSpace(strings.ReplaceAll(norm, "?", " "))
	sig := signals{
		status:         statusPhraseRe.MatchString(plain),
		fineStatus:     fineStatusRe.MatchString(plain),
		request:        requestRe.MatchString(plain),
		ability:        abilityRe.MatchString(plain),
		helpPhrase:     helpPhraseRe.MatchString(plain),
		creative:       creativeRe.MatchString(plain),
		offTopic:       offTopicRe.MatchString(plain),
		injection:      injectionRe.MatchString(plain),
		botFrustration: botFrustrationRe.MatchString(plain),
		cause:          causeRe.MatchString(plain),
		concreteID:     concreteIDRe.MatchString(plain) && !exampleIDRe.MatchString(plain),
		question:       strings.Contains(prompt, "?") || questionStartRe.MatchString(plain),
		onlySocial:     true,
	}
	weak := map[string]bool{}
	for i, tok := range tokens(norm) {
		isSocial := false
		switch {
		case isStrongToken(tok):
			sig.strong = true
		case isWeakToken(tok):
			weak[tok] = true
		case helpWords[tok]:
			sig.help = true
		}
		if isGreetingToken(tok) {
			sig.greeting, isSocial = true, true
		}
		if isThanksToken(tok) {
			sig.thanks, isSocial = true, true
		}
		// The opening "ok"/"grazie" is not evidence that things are fine.
		if positiveWords[tok] && i > 0 {
			sig.positive = true
		}
		if hasAnyPrefix(tok, actionStems) {
			sig.action = true
		}
		if !isSocial && !fillerWords[tok] {
			sig.onlySocial = false
		}
	}
	sig.weak = len(weak)
	return sig
}

func hasAnyPrefix(tok string, stems []string) bool {
	for _, stem := range stems {
		if strings.HasPrefix(tok, stem) {
			return true
		}
	}
	return false
}

func isStrongToken(tok string) bool {
	if strongWords[tok] || statusCodeRe.MatchString(tok) || percentileRe.MatchString(tok) || hasAnyPrefix(tok, strongStems) {
		return true
	}
	return fuzzyIn(tok, typoTargets)
}

func isWeakToken(tok string) bool {
	return weakWords[tok] || hasAnyPrefix(tok, weakStems) || fuzzyIn(tok, weakTypoTargets)
}

func isGreetingToken(tok string) bool {
	return greetingWords[tok] || hasAnyPrefix(tok, greetingStems) || (len(tok) >= 4 && fuzzyInSet(tok, greetingWords))
}

func isThanksToken(tok string) bool {
	return thanksWords[tok] || (len(tok) >= 5 && fuzzyInSet(tok, thanksWords))
}

func fuzzyIn(tok string, targets []string) bool {
	if len(tok) < 5 || commonWords[tok] {
		return false
	}
	for _, target := range targets {
		if withinTypo(tok, target) {
			return true
		}
	}
	return false
}

func fuzzyInSet(tok string, set map[string]bool) bool {
	if commonWords[tok] {
		return false
	}
	for word := range set {
		if len(word) >= 4 && withinTypo(tok, word) {
			return true
		}
	}
	return false
}

// classify decides what the message asks for. The order matters: each rule
// only runs when the earlier, more specific ones did not apply.
func classify(prompt string, scope Scope) intent {
	norm := normalize(prompt)
	toks := tokens(norm)
	if len(toks) == 0 {
		return intentUnclear
	}
	sig := readSignals(prompt, norm)
	window := scope.Explicit || scope.WindowMentioned
	target := scope.Mentions > 0 || window
	// A problem status ("cosa non va", "is prod down") as opposed to a
	// statement that things are fine ("all good").
	badStatus := sig.status && !sig.fineStatus
	concrete := sig.concreteID && (sig.action || sig.request)
	switch {
	case scope.TraceID != "":
		return intentDiagnose
	case sig.injection && !sig.strong:
		return intentUnclear
	case sig.botFrustration && !target:
		return intentUnclear
	case sig.creative:
		return intentUnclear
	case sig.helpPhrase && (!(target && (sig.strong || sig.request)) || exampleMarkerRe.MatchString(norm)):
		return intentHelp
	case sig.thanks && !sig.question && !sig.strong && !sig.request && !badStatus && (!target || sig.positive):
		return intentThanks
	case sig.ability && !sig.status && !sig.cause && !concrete && !window &&
		(scope.Mentions == 0 || !sig.action) && (sig.strong || sig.weak > 0 || sig.action || sig.help || scope.Mentions > 0):
		return intentHelp
	case concrete:
		return intentDiagnose
	case sig.offTopic && !sig.strong:
		return intentUnclear
	case target, sig.strong, sig.status:
		return intentDiagnose
	case sig.weak >= 2, sig.weak >= 1 && len(toks) <= 3, sig.weak >= 1 && sig.request:
		return intentDiagnose
	case sig.help:
		return intentHelp
	case sig.greeting && sig.onlySocial, greetingOnlyRe.MatchString(norm):
		return intentGreeting
	case sig.thanks && sig.onlySocial:
		return intentThanks
	case sig.greeting && len(toks) <= 4 && !sig.offTopic:
		return intentGreeting
	}
	return intentUnclear
}

// "tipo", "for example": what follows illustrates a help question.
var exampleMarkerRe = regexp.MustCompile(`\b(tipo|ad esempio|per esempio|per es|like|for example|for instance|e g|such as)\b`)

// Questions typed without a question mark: "is everything ok".
var questionStartRe = regexp.MustCompile(`^(is|are|do|does|did|can|could|how|what|why|where|when|which|who|ci sono|c e|come|cosa|perche|quanto|quanti|quante|quale|qual|funziona|va)\b`)

var greetingOnlyRe = regexp.MustCompile(`^(what s up|whats up|wassup|come butta|we|ue)\??$`)

// ---------------------------------------------------------------- language

var italianWords = map[string]bool{
	"il": true, "lo": true, "la": true, "gli": true, "le": true, "di": true, "da": true, "con": true,
	"su": true, "per": true, "che": true, "non": true, "sono": true, "ci": true, "ho": true,
	"hai": true, "cosa": true, "come": true, "perche": true, "quando": true, "dove": true,
	"ultimi": true, "ultime": true, "ultima": true, "ultimo": true, "ore": true, "minuti": true,
	"giorni": true, "oggi": true, "ieri": true, "errori": true, "errore": true, "lento": true,
	"sai": true, "puoi": true, "fare": true, "grazie": true, "ciao": true, "questo": true,
	"questa": true, "del": true, "della": true, "dei": true, "delle": true, "nel": true,
	"nella": true, "negli": true, "alla": true, "tutto": true, "va": true, "sta": true,
	"stanno": true, "mi": true, "controlla": true, "succede": true, "problemi": true,
	"servizio": true, "mostrami": true, "dimmi": true, "ora": true, "e": true, "un": true,
	"una": true, "sei": true, "funziona": true, "mezz": true, "settimana": true, "stamattina": true,
	"buongiorno": true, "aiuto": true, "latenza": true, "richieste": true, "tracce": true,
}

var englishWords = map[string]bool{
	"the": true, "of": true, "to": true, "on": true, "for": true, "with": true, "is": true,
	"are": true, "was": true, "what": true, "how": true, "why": true, "when": true, "where": true,
	"last": true, "hours": true, "hour": true, "minutes": true, "days": true, "today": true,
	"yesterday": true, "errors": true, "error": true, "slow": true, "can": true, "you": true,
	"do": true, "does": true, "thanks": true, "hi": true, "hello": true, "this": true, "that": true,
	"any": true, "anything": true, "there": true, "it": true, "my": true, "me": true, "please": true,
	"show": true, "check": true, "past": true, "since": true, "wrong": true, "going": true,
	"happened": true, "up": true, "down": true, "and": true, "an": true, "week": true, "morning": true,
	"help": true, "requests": true, "traces": true, "latency": true, "everything": true,
}

// detectLanguage returns "it" or "en" when the message clearly leans one way,
// "" otherwise (the caller then keeps the UI locale).
func detectLanguage(norm string) string {
	it, en := 0, 0
	for _, tok := range tokens(norm) {
		if italianWords[tok] {
			it++
		}
		if englishWords[tok] {
			en++
		}
	}
	switch {
	case it > en:
		return "it"
	case en > it:
		return "en"
	}
	return ""
}

// ------------------------------------------------------------------- typos

// withinTypo reports whether word is one edit (two for long words) away from
// target, counting an adjacent swap as one edit ("paymnet" -> "payment").
func withinTypo(word, target string) bool {
	limit := 1
	if len(target) >= 8 {
		limit = 2
	}
	return withinEdits(word, target, limit)
}

func withinEdits(word, target string, limit int) bool {
	if word == target {
		return true
	}
	if d := len(word) - len(target); d > limit || -d > limit {
		return false
	}
	return osaDistance(word, target) <= limit
}

func osaDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(rb)]
}

// ------------------------------------------------------------------- scope

// parseScope extracts time window, service and trace id from a free-text prompt.
func parseScope(prompt string, services []string, now time.Time) Scope {
	norm := normalize(prompt)
	scope := Scope{To: now, From: now.Add(-defaultWindow), TraceID: findTraceID(prompt)}
	if window, explicit, mentioned := parseWindow(norm, now); explicit {
		scope.Explicit, scope.WindowMentioned = true, true
		scope.From = now.Add(-window)
		scope.Today = window == sinceMidnight(now) && todayRe.MatchString(norm)
	} else {
		scope.WindowMentioned = mentioned
	}
	found := findServices(pathRe.ReplaceAllString(norm, " "), services)
	scope.Mentions = len(found)
	if len(found) == 1 {
		scope.Service = found[0]
	}
	return scope
}
