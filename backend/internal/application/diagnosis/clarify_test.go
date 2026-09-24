package diagnosis

import (
	"strings"
	"testing"
	"time"
)

// turn runs one chat turn the way Run does, without telemetry: it returns the
// question asked (nil when the analysis would start) and the resulting request.
func turn(t *testing.T, prompt string, prev *Context) (*Report, request) {
	t.Helper()
	return runTurn(prompt, prev)
}

func runTurn(prompt string, prev *Context) (*Report, request) {
	req := understand(prompt, corpusServices, testNow, prev.sanitize(corpusServices))
	if (req.action != actNew && req.action != actMeasure) || req.scope.TraceID != "" {
		return nil, req
	}
	return phrasingFor("it").askMissing(req, corpusServices, prev), req
}

func TestClarificationFlow(t *testing.T) {
	// "diagnostica" names neither window nor service: ask the window first.
	q, _ := turn(t, "diagnostica", nil)
	if q == nil || q.Context.Pending != slotWindow || !strings.Contains(q.Answer, "Su che periodo") || len(q.Suggestions) != 5 {
		t.Fatalf("window question = %+v", q)
	}
	// Picking a window asks for the service, restating what is known.
	q2, _ := turn(t, q.Suggestions[1].Prompt, q.Context)
	if q2 == nil || q2.Context.Pending != slotService || !strings.Contains(q2.Answer, "nell'ultima ora") {
		t.Fatalf("service question = %+v", q2)
	}
	// "tutti i servizi" completes the request: no more questions.
	q3, req := turn(t, "tutti i servizi", q2.Context)
	if q3 != nil || req.action != actNew || req.scope.Service != "" || req.scope.window() != time.Hour {
		t.Fatalf("completed request = %+v / %+v", q3, req)
	}

	// A measure with its service but no window asks only the window.
	m, _ := turn(t, "voglio sapere in media la velocità di tutte le GET del catalogo", nil)
	if m == nil || m.Context.Pending != slotWindow || m.Context.Measure == nil || m.Context.Measure.Method != "GET" || m.Context.Service != "catalog-service" {
		t.Fatalf("measure question = %+v", m)
	}
	if !strings.Contains(m.Answer, "latenza media per le chiamate GET") {
		t.Errorf("measure recap = %q", m.Answer)
	}
	m2, req := turn(t, "2 ore", m.Context)
	if m2 != nil || req.action != actMeasure || req.scope.Service != "catalog-service" || req.scope.window() != 2*time.Hour || req.measure.Stat != statAvg {
		t.Fatalf("completed measure = %+v / %+v", m2, req)
	}

	// Everything stated up front: no question at all.
	if q, req := turn(t, "errori di payment nelle ultime 2 ore", nil); q != nil || req.scope.Service != "payment-service" {
		t.Fatalf("complete request asked %+v", q)
	}

	// Two services: ask which one, then continue with the chosen one.
	two, _ := turn(t, "confronta payment e checkout nelle ultime 2 ore", nil)
	if two == nil || two.Context.Pending != slotService || len(two.Context.Candidates) != 2 {
		t.Fatalf("which service question = %+v", two)
	}
	if q, req := turn(t, "payment", two.Context); q != nil || req.scope.Service != "payment-service" || req.scope.window() != 2*time.Hour {
		t.Fatalf("after choosing = %+v / %+v", q, req)
	}

	// "fai tu" takes the default window, then asks the service.
	q, _ = turn(t, "ci sono errori?", nil)
	if d, _ := turn(t, "fai tu", q.Context); d == nil || d.Context.Pending != slotService {
		t.Fatalf("delegated window = %+v", d)
	}

	// A message that does not answer the question starts over.
	if _, req := turn(t, "cosa sai fare?", q.Context); req.action != actReply || req.intent != intentHelp {
		t.Fatalf("unrelated message during a question = %+v", req)
	}
}
