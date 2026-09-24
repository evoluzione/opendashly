package diagnosis

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// clarifyCase is one labeled message from testdata/clarify_corpus.json:
// an optional pending question, the message and what should happen.
type clarifyCase struct {
	Prev *struct {
		Pending    string   `json:"pending"`
		Focus      string   `json:"focus"`
		Service    string   `json:"service"`
		ServiceSet bool     `json:"service_set"`
		Window     int      `json:"window_minutes"`
		WindowSet  bool     `json:"window_set"`
		Measure    *Measure `json:"measure"`
	} `json:"prev"`
	Prompt  string  `json:"prompt"`
	Action  string  `json:"action"`
	Service *string `json:"service"`
	Window  *int    `json:"window_minutes"`
	Metric  *string `json:"metric"`
	Stat    *string `json:"stat"`
	Method  *string `json:"method"`
	Path    *string `json:"path"`
	Why     string  `json:"why"`
	Lens    string  `json:"lens"`
}

func (c clarifyCase) context(now time.Time) *Context {
	if c.Prev == nil {
		return nil
	}
	w := time.Duration(c.Prev.Window) * time.Minute
	today := c.Prev.Window == -1
	if today {
		w = sinceMidnight(now)
	} else if w <= 0 {
		w = defaultWindow
	}
	ctx := &Context{From: now.Add(-w), To: now, Today: today, Service: c.Prev.Service, Focus: c.Prev.Focus,
		Pending: c.Prev.Pending, WindowSet: c.Prev.WindowSet, ServiceSet: c.Prev.ServiceSet, Measure: c.Prev.Measure}
	if ctx.Measure != nil && ctx.Measure.Stat == "null" {
		ctx.Measure.Stat = ""
	}
	return ctx
}

func checkClarify(c clarifyCase, now time.Time) []string {
	q, req := runTurn(c.Prompt, c.context(now))
	got := ""
	scope := req.scope
	switch {
	case q != nil && q.Context.Pending == slotWindow:
		got = "ask_window"
	case q != nil && len(q.Context.Candidates) > 1:
		got = "ask_which_service"
	case q != nil:
		got = "ask_service"
	case req.action == actReply:
		got = intentNames[req.intent]
	case req.action == actMeasure:
		got = "measure"
	case req.action == actNew || req.action == actFollowUp || req.action == actDetails:
		got = "new"
	case req.action == actUnsupported:
		got = "unsupported"
	case req.action == actLinks:
		got = "links"
	default:
		got = "unclear" // references with nothing to point at
	}
	var diffs []string
	if got != c.Action {
		return []string{fmt.Sprintf("action=%s want %s", got, c.Action)}
	}
	if q != nil {
		scope = Scope{From: q.Context.From, To: q.Context.To, Service: q.Context.Service}
	}
	if c.Service != nil && scope.Service != *c.Service {
		diffs = append(diffs, fmt.Sprintf("service=%q want %q", scope.Service, *c.Service))
	}
	if c.Window != nil && got != "ask_window" && (got == "new" || got == "measure" || got == "ask_service" || got == "ask_which_service") {
		want := *c.Window
		if want == -1 {
			want = int(sinceMidnight(now).Minutes())
		}
		if want > int(maxWindow.Minutes()) {
			want = int(maxWindow.Minutes())
		}
		if gotMin := int(scope.window().Minutes()); gotMin != want {
			diffs = append(diffs, fmt.Sprintf("window=%d want %d", gotMin, want))
		}
	}
	m := req.measure
	if q != nil && q.Context.Measure != nil {
		m = *q.Context.Measure
	}
	if got == "measure" || (q != nil && q.Context.Measure != nil) {
		if c.Metric != nil && m.Metric != *c.Metric {
			diffs = append(diffs, fmt.Sprintf("metric=%q want %q", m.Metric, *c.Metric))
		}
		if c.Stat != nil && *c.Stat != "" && m.Stat != *c.Stat {
			diffs = append(diffs, fmt.Sprintf("stat=%q want %q", m.Stat, *c.Stat))
		}
		if c.Method != nil && m.Method != *c.Method {
			diffs = append(diffs, fmt.Sprintf("method=%q want %q", m.Method, *c.Method))
		}
		if c.Path != nil && m.Path != *c.Path {
			diffs = append(diffs, fmt.Sprintf("path=%q want %q", m.Path, *c.Path))
		}
	} else if c.Metric != nil && *c.Metric != "" && (c.Action == "ask_window" || c.Action == "ask_service") {
		diffs = append(diffs, fmt.Sprintf("measure not detected, want metric %q", *c.Metric))
	}
	return diffs
}

func TestClarifyCorpus(t *testing.T) {
	raw, err := os.ReadFile("testdata/clarify_corpus.json")
	if err != nil {
		t.Skip("no clarify corpus")
	}
	var cases []clarifyCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	failures := map[string][]string{}
	failed := 0
	for _, c := range cases {
		if diffs := checkClarify(c, testNow); diffs != nil {
			failed++
			prev := "no prev"
			if c.Prev != nil {
				prev = fmt.Sprintf("pending=%s svc=%q set=%v win=%d set=%v", c.Prev.Pending, c.Prev.Service, c.Prev.ServiceSet, c.Prev.Window, c.Prev.WindowSet)
			}
			failures[c.Lens] = append(failures[c.Lens], fmt.Sprintf("%q [%s]: %s (%s)", c.Prompt, prev, strings.Join(diffs, ", "), c.Why))
		}
	}
	lenses := make([]string, 0, len(failures))
	for l := range failures {
		lenses = append(lenses, l)
	}
	sort.Strings(lenses)
	for _, l := range lenses {
		for _, f := range failures[l] {
			t.Errorf("[%s] %s", l, f)
		}
	}
	if failed > 0 {
		t.Logf("%d/%d clarify cases failed", failed, len(cases))
	}
}
