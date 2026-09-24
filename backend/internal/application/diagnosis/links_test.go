package diagnosis

import (
	"context"
	"testing"
	"time"
)

func TestLinks(t *testing.T) {
	r := &Runner{}
	ctx := context.Background()

	// An analysis links only to data it found: nothing here, no telemetry.
	rep, err := r.Run(ctx, "errori su tutti i servizi nelle ultime 2 ore", "it", testNow, nil)
	if err != nil || len(rep.Links) != 0 {
		t.Fatalf("analysis links without data = %+v, %v", rep.Links, err)
	}
	two := testNow.Add(-2 * time.Hour)
	links2 := phrasingFor("it").focusLinks(Scope{From: two, To: testNow}, focusErrors, true, true)
	if len(links2) != 2 {
		t.Fatalf("analysis links = %+v", links2)
	}
	logs := links2[0]
	if logs.Kind != linkLogs || logs.Service != "" || logs.Severity != errorSeverities || logs.To.Sub(logs.From) != 2*time.Hour {
		t.Errorf("error logs link = %+v", logs)
	}
	if tr := links2[1]; tr.Kind != linkTraces || !tr.ErrorsOnly {
		t.Errorf("error traces link = %+v", tr)
	}

	// "scaricami i log" after an answer: the same scope, logs only, no questions.
	prev := &Context{From: testNow.Add(-3 * time.Hour), To: testNow, Service: "cart-service", Focus: focusErrors}
	req := understand("scaricami i log", corpusServices, testNow, prev.sanitize(corpusServices))
	if req.action != actLinks || !req.linkLogs || req.linkTraces || req.scope.Service != "cart-service" || req.scope.window() != 3*time.Hour {
		t.Fatalf("links follow-up = %+v", req)
	}
	links := phrasingFor("it").linksReport(req, prev)
	if len(links.Links) != 1 || links.Links[0].Kind != linkLogs || links.Links[0].Severity != errorSeverities {
		t.Errorf("links report = %+v", links.Links)
	}

	// Without a previous answer the window is asked first, and remembered.
	q, _ := runTurn("dammi il link alle trace di payment", nil)
	if q == nil || q.Context.Pending != slotWindow || q.Context.Want != wantLinks {
		t.Fatalf("links question = %+v", q)
	}
	if _, done := runTurn("ultima ora", q.Context); done.action != actLinks || !done.linkTraces {
		t.Fatalf("links after the window = %+v", done)
	}

	// A trace answer links to the trace itself.
	tr, _ := r.Run(ctx, "4bf92f3577b34da6a3ce929d0e0e4736", "it", testNow, nil)
	if len(tr.Links) != 1 || tr.Links[0].Kind != linkTrace || tr.Links[0].TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace links = %+v", tr.Links)
	}
}

func TestLinksErrorsOnlyWhenAboutErrors(t *testing.T) {
	p := phrasingFor("it")
	// A fresh request for traces, no errors mentioned: all traces.
	req := understand("dammi il link alle trace di payment dell ultima ora", corpusServices, testNow, nil)
	if l := p.linksReport(req, nil).Links; len(l) != 1 || l[0].ErrorsOnly {
		t.Errorf("fresh traces link = %+v", l)
	}
	// After a diagnosis, the same words mean the error traces it talked about.
	prev := &Context{From: testNow.Add(-time.Hour), To: testNow, Service: "payment-service"}
	req = understand("dammi il link alle trace", corpusServices, testNow, prev.sanitize(corpusServices))
	if l := p.linksReport(req, prev).Links; len(l) != 1 || !l[0].ErrorsOnly {
		t.Errorf("follow-up traces link = %+v", l)
	}
	// "tutti i log" overrides it.
	req = understand("scarica tutti i log", corpusServices, testNow, prev.sanitize(corpusServices))
	if l := p.linksReport(req, prev).Links; len(l) != 1 || l[0].Severity != "" {
		t.Errorf("all logs link = %+v", l)
	}
}
