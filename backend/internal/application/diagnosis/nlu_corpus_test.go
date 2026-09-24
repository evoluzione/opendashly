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

// corpusCase is one labeled chat message from testdata/nlu_corpus.json.
// window_minutes: >0 explicit minutes, 0 no window, -1 since midnight, null any.
// service / trace_id: "" none, null any.
type corpusCase struct {
	Prompt  string   `json:"prompt"`
	Lang    string   `json:"lang"`
	Intents []string `json:"intents"`
	Window  *int     `json:"window_minutes"`
	Service *string  `json:"service"`
	TraceID *string  `json:"trace_id"`
	Why     string   `json:"why"`
	Lens    string   `json:"lens"`
}

var corpusServices = []string{
	"api-gateway", "auth-service", "cart-service", "catalog-service", "checkout-service",
	"inventory-service", "notification-service", "payment-service", "shipping-service",
}

var intentNames = map[intent]string{
	intentDiagnose: "diagnose", intentHelp: "help", intentGreeting: "greeting",
	intentThanks: "thanks", intentUnclear: "unclear",
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/nlu_corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []corpusCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

// checkCase returns what differs from the label, or nil.
func checkCase(c corpusCase, now time.Time) []string {
	scope := parseScope(c.Prompt, corpusServices, now)
	got := intentNames[classify(c.Prompt, scope)]
	var diffs []string
	ok := false
	for _, want := range c.Intents {
		if want == got {
			ok = true
		}
	}
	if !ok {
		diffs = append(diffs, fmt.Sprintf("intent=%s want %v", got, c.Intents))
	}
	if c.Window != nil {
		gotMin := 0
		if scope.Explicit {
			gotMin = int(scope.window().Minutes())
		}
		want := *c.Window
		if want == -1 {
			y, mo, d := now.Date()
			want = int(now.Sub(time.Date(y, mo, d, 0, 0, 0, 0, now.Location())).Minutes())
		}
		if want > int(maxWindow.Minutes()) {
			want = int(maxWindow.Minutes())
		}
		if gotMin != want {
			diffs = append(diffs, fmt.Sprintf("window=%d want %d", gotMin, want))
		}
	}
	if c.Service != nil && scope.Service != *c.Service {
		diffs = append(diffs, fmt.Sprintf("service=%q want %q", scope.Service, *c.Service))
	}
	if c.TraceID != nil && scope.TraceID != strings.ToLower(*c.TraceID) {
		diffs = append(diffs, fmt.Sprintf("trace=%q want %q", scope.TraceID, *c.TraceID))
	}
	return diffs
}

func TestNLUCorpus(t *testing.T) {
	cases := loadCorpus(t)
	failures := map[string][]string{}
	failed := 0
	for _, c := range cases {
		if diffs := checkCase(c, testNow); diffs != nil {
			failed++
			failures[c.Lens] = append(failures[c.Lens], fmt.Sprintf("%q: %s (%s)", c.Prompt, strings.Join(diffs, ", "), c.Why))
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
		t.Logf("%d/%d corpus cases failed", failed, len(cases))
	}
}
