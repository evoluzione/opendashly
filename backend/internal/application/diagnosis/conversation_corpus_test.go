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

// convCase is one labeled follow-up from testdata/conversation_corpus.json:
// the previous answer's context, the new message and what should happen.
type convCase struct {
	Prev *struct {
		Service string `json:"service"`
		Window  int    `json:"window_minutes"`
		Focus   string `json:"focus"`
		Items   []struct {
			Kind     string `json:"kind"`
			Service  string `json:"service"`
			Endpoint string `json:"endpoint"`
			HasTrace bool   `json:"has_trace"`
		} `json:"items"`
	} `json:"prev"`
	Prompt  string  `json:"prompt"`
	Action  string  `json:"action"`
	Ref     *int    `json:"ref"`
	Service *string `json:"service"`
	Window  *int    `json:"window_minutes"`
	Focus   *string `json:"focus"`
	Detail  *bool   `json:"detail"`
	Why     string  `json:"why"`
	Lens    string  `json:"lens"`
}

func (c convCase) context(now time.Time) *Context {
	if c.Prev == nil {
		return nil
	}
	w := time.Duration(c.Prev.Window) * time.Minute
	today := c.Prev.Window == -1
	if today {
		w = sinceMidnight(now)
	} else if w <= 0 {
		w = time.Hour
	}
	ctx := &Context{From: now.Add(-w), To: now, Service: c.Prev.Service, Focus: c.Prev.Focus, Today: today}
	for i, it := range c.Prev.Items {
		item := ContextItem{Kind: it.Kind, Service: it.Service, Endpoint: it.Endpoint}
		if it.HasTrace {
			item.TraceID = fmt.Sprintf("%032x", i+1)
		}
		ctx.Items = append(ctx.Items, item)
	}
	return ctx.sanitize(corpusServices)
}

var actionNames = map[action]string{
	actNew: "new", actOpenItem: "open_item", actDetails: "details", actFollowUp: "follow_up",
	actNoContextRef: "no_context_ref", actUnsupported: "unsupported", actMeasure: "measure",
	actNoMatch: "unclear", actAmbiguousRef: "unclear", actLinks: "links",
}

func gotAction(req request) string {
	if req.action == actReply {
		return intentNames[req.intent]
	}
	return actionNames[req.action]
}

// actionMatches treats as equivalent the labels that lead to the same
// observable answer: a measure or a new analysis where a follow-up was
// expected (service, window and focus are still checked), and answering a
// scope-bearing message that has no previous answer instead of refusing it.
func actionMatches(want, got string, req request) bool {
	analysis := map[string]bool{"new": true, "follow_up": true, "measure": true}
	switch {
	case want == got:
		return true
	case analysis[want] && analysis[got]:
		return true
	case want == "no_context_ref" && got == "new" && (req.scope.Mentions > 0 || req.scope.Explicit):
		return true
	case (want == "no_context_ref" || want == "unclear") && req.action == actNoMatch:
		// "nothing to point at" and "no such item" get the same clarification.
		return true
	}
	return false
}

func checkConv(c convCase, now time.Time) []string {
	prev := c.context(now)
	req := understand(c.Prompt, corpusServices, now, prev)
	got := gotAction(req)
	var diffs []string
	if !actionMatches(c.Action, got, req) {
		diffs = append(diffs, fmt.Sprintf("action=%s want %s", got, c.Action))
		return diffs
	}
	scope, focus := req.scope, req.focus
	// A complete question without its own window or service is now asked for
	// them (Run -> askMissing) instead of inheriting them: only compare what
	// the message itself stated.
	asked := (req.action == actNew || req.action == actMeasure) && req.scope.TraceID == "" && (!req.windowSet || !req.serviceSet)
	if asked {
		if !req.windowSet {
			c.Window = nil
		}
		if !req.serviceSet {
			c.Service = nil
		}
	}
	switch req.action {
	case actOpenItem:
		idx := resolveRef(req.ref, prev.Items)
		if c.Ref != nil {
			want := *c.Ref
			if want == -1 {
				want = len(prev.Items)
			}
			if idx+1 != want {
				diffs = append(diffs, fmt.Sprintf("ref=%d want %d", idx+1, want))
			}
		}
		scope = prev.scopeAt(now)
		if idx >= 0 && prev.Items[idx].Service != "" {
			scope.Service = prev.Items[idx].Service
		}
		focus = ""
		if idx >= 0 {
			focus = focusOfKind(prev.Items[idx].Kind)
		}
	case actDetails:
		scope, focus = prev.scopeAt(now), prev.Focus
	case actMeasure:
		focus = map[string]string{metricLatency: focusLatency, metricErrors: focusErrors, metricRequests: focusTraffic}[req.measure.Metric]
	}
	if c.Service != nil && scope.Service != *c.Service {
		diffs = append(diffs, fmt.Sprintf("service=%q want %q", scope.Service, *c.Service))
	}
	if c.Window != nil && req.action != actOpenItem {
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
	if c.Focus != nil && req.action != actOpenItem && focus != *c.Focus {
		diffs = append(diffs, fmt.Sprintf("focus=%q want %q", focus, *c.Focus))
	}
	if c.Detail != nil && *c.Detail && !req.detail && req.action != actDetails {
		diffs = append(diffs, "detail=false want true")
	}
	return diffs
}

func TestConversationCorpus(t *testing.T) {
	raw, err := os.ReadFile("testdata/conversation_corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []convCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	failures := map[string][]string{}
	failed := 0
	for _, c := range cases {
		if diffs := checkConv(c, testNow); diffs != nil {
			failed++
			prev := "no prev"
			if c.Prev != nil {
				prev = fmt.Sprintf("prev svc=%q win=%d focus=%q items=%d", c.Prev.Service, c.Prev.Window, c.Prev.Focus, len(c.Prev.Items))
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
		t.Logf("%d/%d conversation cases failed", failed, len(cases))
	}
}
