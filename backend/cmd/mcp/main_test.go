package main

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToolCallLogsInOnUnauthorized(t *testing.T) {
	var gotQuery map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			http.SetCookie(w, &http.Cookie{Name: "oteldash_session", Value: "ok", Path: "/"})
			return
		}
		if c, err := r.Cookie("oteldash_session"); err != nil || c.Value != "ok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotQuery)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	c := &client{base: srv.URL, http: &http.Client{Jar: jar}}
	result, rpcErr := c.handle(rpcRequest{
		ID:     json.RawMessage(`1`),
		Method: "tools/call",
		Params: json.RawMessage(`{"name":"query_telemetry","arguments":{"signals":["logs"],"filters":[{"key":"severity","operator":"=","value":"ERROR"}]}}`),
	})
	if rpcErr != nil {
		t.Fatalf("rpc error: %v", *rpcErr)
	}
	res := result.(map[string]any)
	if res["isError"] == true {
		t.Fatalf("tool error: %v", res["content"])
	}
	if !strings.Contains(res["content"].([]map[string]any)[0]["text"].(string), `"ok"`) {
		t.Fatalf("unexpected result: %v", res)
	}
	f := gotQuery["filterList"].([]any)[0].(map[string]any)
	if f["connector"] != "AND" || gotQuery["limit"].(float64) != 50 {
		t.Fatalf("unexpected query: %v", gotQuery)
	}
}
