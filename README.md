<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/opendashly-logo-dark.svg">
  <img src="assets/opendashly-logo-light.svg" alt="Opendashly" height="48">
</picture>

Self-hosted observability for logs, traces and metrics. Receives OpenTelemetry over OTLP, stores it in ClickHouse, serves it through a fast dashboard.

```mermaid
flowchart LR
    A(["Your services"]):::app -->|OTLP| B["Collector"]:::infra
    B --> C[("ClickHouse")]:::db
    C --> D["Go API"]:::api
    D --> E(["Dashboard"]):::ui

    classDef app  fill:#6c5ce7,stroke:#a29bfe,color:#fff
    classDef infra fill:#00cec9,stroke:#55efc4,color:#fff
    classDef db   fill:#fdcb6e,stroke:#e17055,color:#111
    classDef api  fill:#fd79a8,stroke:#e84393,color:#fff
    classDef ui   fill:#6c5ce7,stroke:#a29bfe,color:#fff
```

- Trace correlation: from a log line to its trace in one click
- Built-in diagnosis assistant, no LLM or API key
- Configurable data retention
- JWT auth with roles
- Whole stack in one `docker compose up`

## Quick start

Requires Docker 20.10+ and Docker Compose 2.0+.

```bash
docker compose up --build
```

| Service     | URL                   |
| ----------- | --------------------- |
| Dashboard   | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| OTLP HTTP   | http://localhost:4318 |
| OTLP gRPC   | grpc://localhost:4317 |
| ClickHouse  | http://localhost:8123 |

Log in with `admin` / `admin`; you'll be asked to change the password.

Point any OpenTelemetry SDK at `http://localhost:4318`, or send a test log:

```bash
curl -X POST http://localhost:4318/v1/logs -H "Content-Type: application/json" -d '{
  "resourceLogs": [{
    "resource": { "attributes": [{ "key": "service.name", "value": { "stringValue": "my-service" } }] },
    "scopeLogs": [{ "logRecords": [{ "severityText": "INFO", "body": { "stringValue": "hello" } }] }]
  }]
}'
```

Generate realistic ecommerce traffic (logs, traces, metrics across 9 services):

```bash
npm --prefix scripts install
node scripts/loadtest_otel_node.mjs --duration 300 --rps 100
```

Other flags: `--services`, `--error-rate`, `--hot-rate`, `--no-metrics`, `--no-traces`, `--collector`.

## Diagnosis

Ask in Italian or English, e.g. `checkout-service ultime 2 ore`, `p95 di /checkout`, or paste a trace ID. It compares against the previous window and reports error rate, latency regressions, error hotspots and where failures start. Runs inside the backend, no external service. Test corpora live in `backend/internal/application/diagnosis/testdata/`.

## Claude (MCP)

Read-only MCP server (`query_telemetry`, `get_trace`, `get_trace_related`, `list_services`), shipped as `ghcr.io/evoluzione/opendashly-mcp`. Use a dedicated non-admin user that has already changed its first-login password.

```bash
claude mcp add opendashly --scope user -- docker run -i --rm --pull always \
  -e OPENDASHLY_URL=https://opendashly.example.com \
  -e OPENDASHLY_USERNAME=claude \
  -e OPENDASHLY_PASSWORD='...' \
  ghcr.io/evoluzione/opendashly-mcp
```

For Claude Desktop, put the same `docker run` command and args under `mcpServers.opendashly` in `claude_desktop_config.json`. If Opendashly runs on your machine, use `http://host.docker.internal:8080` as the URL. Without Docker: `go build ./cmd/mcp` in `backend/`.

## Production

Uses prebuilt GHCR images; you only need `docker-compose.prod.yml` and a `.env` next to it:

```bash
AUTH_SECRET=replace-with-32-plus-random-chars
CLICKHOUSE_PASSWORD=replace-with-strong-password
CORS_ALLOWED_ORIGINS=http://your-host:5173
```

```bash
docker compose -f docker-compose.prod.yml up -d
```

Minimum 1 vCPU / 2 GB RAM / 20 GB SSD, recommended 4 vCPU / 8 GB / 50 GB+. No tuning variables: backend, collector and ClickHouse size themselves from available CPU and RAM and back off under pressure.

Before exposing it: change the admin password, serve over HTTPS, keep ClickHouse ports (8123, 9000) private, restrict `CORS_ALLOWED_ORIGINS`, set up backups.

---

Built on [OpenTelemetry](https://opentelemetry.io), [ClickHouse](https://clickhouse.com), [Go](https://go.dev), [SvelteKit](https://kit.svelte.dev), [uPlot](https://github.com/leeoniya/uPlot), [Chi](https://github.com/go-chi/chi).
