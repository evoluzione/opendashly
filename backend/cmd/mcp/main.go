// Command mcp is a stdio MCP server that lets Claude read logs and traces
// from a running OpenDashly instance through its public HTTP API.
//
// Env: OPENDASHLY_URL, OPENDASHLY_USERNAME, OPENDASHLY_PASSWORD.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxToolOutput = 80000

type client struct {
	base, user, pass string
	http             *http.Client
}

func (c *client) login() error {
	body, _ := json.Marshal(map[string]string{"username": c.user, "password": c.pass})
	resp, err := c.http.Post(c.base+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s %s", resp.Status, msg)
	}
	return nil
}

// do calls the API, logging in again once if the session cookie expired.
func (c *client) do(method, path string, payload any) (string, error) {
	for attempt := 0; ; attempt++ {
		var body io.Reader
		if payload != nil {
			b, _ := json.Marshal(payload)
			body = bytes.NewReader(b)
		}
		req, err := http.NewRequest(method, c.base+path, body)
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return "", err
		}
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			if err := c.login(); err != nil {
				return "", err
			}
			continue
		}
		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("%s %s: %s %s", method, path, resp.Status, out)
		}
		return string(out), nil
	}
}

var tools = []map[string]any{
	{
		"name":        "query_telemetry",
		"description": "Search OpenDashly logs and/or traces in a time window. Filter keys: service.name, trace_id, span_name, severity (TRACE/DEBUG/INFO/WARN/ERROR/FATAL), body, duration_ms, trace_error_scope, or any OTel attribute key. Operators: =, !=, >, <, >=, <=, contains.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"signals":     map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"logs", "traces"}}, "description": "Default: [\"logs\",\"traces\"]"},
				"lastMinutes": map[string]any{"type": "integer", "description": "Window ending now. Default 60. Ignored if from/to are set."},
				"from":        map[string]any{"type": "string", "description": "RFC3339 start"},
				"to":          map[string]any{"type": "string", "description": "RFC3339 end"},
				"filters": map[string]any{"type": "array", "items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"key": map[string]any{"type": "string"}, "operator": map[string]any{"type": "string"},
						"value": map[string]any{"type": "string"}, "connector": map[string]any{"type": "string", "enum": []string{"AND", "OR"}},
					},
					"required": []string{"key", "operator", "value"},
				}},
				"limit": map[string]any{"type": "integer", "description": "Rows per signal, default 50"},
				"page":  map[string]any{"type": "integer", "description": "1-based page, default 1"},
			},
		},
	},
	{
		"name":        "get_trace",
		"description": "Get all spans of a trace by trace ID.",
		"inputSchema": traceIDSchema,
	},
	{
		"name":        "get_trace_related",
		"description": "Get logs and telemetry correlated with a trace ID.",
		"inputSchema": traceIDSchema,
	},
	{
		"name":        "list_services",
		"description": "List service names known to OpenDashly.",
		"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
	},
}

var traceIDSchema = map[string]any{
	"type":       "object",
	"properties": map[string]any{"traceId": map[string]any{"type": "string"}},
	"required":   []string{"traceId"},
}

type queryArgs struct {
	Signals     []string         `json:"signals"`
	LastMinutes int              `json:"lastMinutes"`
	From        string           `json:"from"`
	To          string           `json:"to"`
	Filters     []map[string]any `json:"filters"`
	Limit       int              `json:"limit"`
	Page        int              `json:"page"`
}

func (c *client) call(name string, raw json.RawMessage) (string, error) {
	switch name {
	case "query_telemetry":
		var a queryArgs
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
		}
		if len(a.Signals) == 0 {
			a.Signals = []string{"logs", "traces"}
		}
		if a.LastMinutes <= 0 {
			a.LastMinutes = 60
		}
		if a.Limit <= 0 {
			a.Limit = 50
		}
		if a.Page <= 0 {
			a.Page = 1
		}
		to := time.Now().UTC()
		from := to.Add(-time.Duration(a.LastMinutes) * time.Minute)
		if a.From != "" || a.To != "" {
			var err error
			if from, err = time.Parse(time.RFC3339, a.From); err != nil {
				return "", fmt.Errorf("from: %w", err)
			}
			if to, err = time.Parse(time.RFC3339, a.To); err != nil {
				return "", fmt.Errorf("to: %w", err)
			}
		}
		for _, f := range a.Filters {
			if f["connector"] == nil {
				f["connector"] = "AND"
			}
		}
		return c.do(http.MethodPost, "/api/query/run", map[string]any{
			"signals":    a.Signals,
			"timeRange":  map[string]any{"from": from, "to": to},
			"filterList": a.Filters,
			"limit":      a.Limit,
			"page":       a.Page,
		})
	case "get_trace", "get_trace_related":
		var a struct {
			TraceID string `json:"traceId"`
		}
		if err := json.Unmarshal(raw, &a); err != nil || a.TraceID == "" {
			return "", fmt.Errorf("traceId is required")
		}
		suffix := "/spans"
		if name == "get_trace_related" {
			suffix = "/related"
		}
		return c.do(http.MethodGet, "/api/traces/"+url.PathEscape(a.TraceID)+suffix, nil)
	case "list_services":
		return c.do(http.MethodGet, "/api/services", nil)
	}
	return "", fmt.Errorf("unknown tool %q", name)
}

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func (c *client) handle(req rpcRequest) (any, *map[string]any) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(req.Params, &p)
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = "2025-06-18"
		}
		return map[string]any{
			"protocolVersion": p.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "opendashly", "version": "1.0.0"},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": tools}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &map[string]any{"code": -32602, "message": err.Error()}
		}
		text, err := c.call(p.Name, p.Arguments)
		if err != nil {
			return map[string]any{"content": []map[string]any{{"type": "text", "text": err.Error()}}, "isError": true}, nil
		}
		if len(text) > maxToolOutput {
			text = text[:maxToolOutput] + "\n...[truncated: narrow the time range, add filters or lower limit]"
		}
		return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}, nil
	}
	return nil, &map[string]any{"code": -32601, "message": "method not found: " + req.Method}
}

func main() {
	log.SetOutput(os.Stderr) // stdout is reserved for JSON-RPC
	base := strings.TrimRight(os.Getenv("OPENDASHLY_URL"), "/")
	if base == "" {
		log.Fatal("OPENDASHLY_URL is required")
	}
	jar, _ := cookiejar.New(nil)
	c := &client{
		base: base,
		user: os.Getenv("OPENDASHLY_USERNAME"),
		pass: os.Getenv("OPENDASHLY_PASSWORD"),
		http: &http.Client{Jar: jar, Timeout: 60 * time.Second},
	}

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 16<<20)
	out := json.NewEncoder(os.Stdout)
	for in.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(in.Bytes(), &req); err != nil {
			out.Encode(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32700, "message": err.Error()}})
			continue
		}
		if len(req.ID) == 0 {
			continue // notification
		}
		result, rpcErr := c.handle(req)
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if rpcErr != nil {
			resp["error"] = *rpcErr
		} else {
			resp["result"] = result
		}
		out.Encode(resp)
	}
}
