<div align="center">

<div style="display: inline-flex; align-items: center; gap: 24px; text-align: left;">
  <picture style="flex: 0 0 auto; display: block; line-height: 0;">
    <source media="(prefers-color-scheme: dark)" srcset="assets/opendashly-mark-dark.svg" />
    <source media="(prefers-color-scheme: light)" srcset="assets/opendashly-mark-light.svg" />
    <img src="assets/opendashly-mark-light.svg" alt="OpenDashly logo" width="96" height="96" style="display: block;" />
  </picture>
  <div style="display: flex; flex-direction: column; justify-content: center; line-height: 1; transform: translateY(-2px);">
    <strong style="font-size: 2.45em; line-height: 0.95; letter-spacing: -0.03em; color: #f8fafc;">Opendashly</strong>
    <span style="margin-top: 8px; font-size: 1em; letter-spacing: 0.12em; text-transform: uppercase; color: #64748b; line-height: 1;">Dashboard</span>
  </div>
</div>

**Complete observability for OpenTelemetry-instrumented services — logs, traces, and metrics in one self-hosted platform.**

[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.20-6c5ce7?style=flat-square&logo=opentelemetry&logoColor=white)](https://opentelemetry.io)
[![ClickHouse](https://img.shields.io/badge/ClickHouse-24-FFCC01?style=flat-square&logo=clickhouse&logoColor=black)](https://clickhouse.com)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-4.2-FF3E00?style=flat-square&logo=svelte&logoColor=white)](https://kit.svelte.dev)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker&logoColor=white)](https://docker.com)

[Quick Start](#-quick-start) · [AI Features](#-ai-powered-queries) · [Production Deploy](#-production-deployment) · [Tech Stack](#-tech-stack)

</div>

---

## What is Opendashly?

Opendashly sits between your services and your team. It receives telemetry over standard OTLP endpoints, stores it in ClickHouse, and serves it through a fast, interactive dashboard.

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

**Why Opendashly?**

- **High-throughput ingestion** — ClickHouse columnar storage handles millions of events/second with sub-second queries
- **Trace correlation** — navigate from a log line to its trace and spans in one click
- **AI observability agent** — ask questions in plain language, the agent investigates your telemetry autonomously and surfaces problems, anomalies, and insights
- **Single-command deploy** — the entire stack runs with `docker compose up`
- **Data retention policies** — configurable automatic cleanup keeps storage costs predictable
- **Enterprise auth** — JWT with role-based access control and bcrypt password hashing

---

## ⚡ Quick Start

**Prerequisites:** Docker 20.10+ and Docker Compose 2.0+. Nothing else.

### 1. Start the stack

```bash
docker compose up --build
```

| Service | URL |
|---|---|
| Dashboard | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| OTLP HTTP | http://localhost:4318 |
| OTLP gRPC | grpc://localhost:4317 |
| ClickHouse | http://localhost:8123 |

### 2. Login

```
Username: admin
Password: admin
```

> ⚠️ You will be prompted to change your password on first login.

### 3. Send your first log

```bash
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{ "key": "service.name", "value": { "stringValue": "my-service" }}]
      },
      "scopeLogs": [{
        "logRecords": [{
          "severityText": "INFO",
          "body": { "stringValue": "Hello from OpenTelemetry!" }
        }]
      }]
    }]
  }'
```

Or point any OpenTelemetry-instrumented application at `http://localhost:4318` (HTTP) or `grpc://localhost:4317` (gRPC).

### 4. Generate realistic traffic (optional)

The repository includes a Node.js load generator that simulates ecommerce microservices with consistent logs, traces, and metrics.

```bash
# Install once
npm --prefix scripts install

# Run for 5 minutes at 100 req/s
node scripts/loadtest_otel_node.mjs --duration 300 --rps 100
```

Services simulated: `api-gateway`, `auth-service`, `catalog-service`, `cart-service`, `checkout-service`, `payment-service`, `inventory-service`, `shipping-service`, `notification-service`.

<details>
<summary>Load test options</summary>

| Flag | Description | Default |
|---|---|---|
| `--duration` | Test duration in seconds | — |
| `--rps` | Target requests per second | — |
| `--services` | Active microservices (2–9) | 9 |
| `--error-rate` | Base error probability (0–1) | 0.02 |
| `--hot-rate` | Extra traffic on checkout hotspot (0–1) | 0.2 |
| `--no-metrics` | Disable metrics signal | — |
| `--no-traces` | Disable traces signal | — |
| `--collector` | OTLP HTTP base URL | `http://localhost:4318` |

</details>

---

## 🤖 AI Observability Agent

Opendashly ships a built-in AI agent powered by OpenAI. Ask a question in plain language — in Italian or English — and the agent takes it from there: it runs queries against your telemetry, identifies anomalies and error patterns, and returns results directly as log tables, trace lists, and actionable insights. No SQL required, no manual filtering.

**Setup:** Settings → AI Configuration → enable, paste your OpenAI API key, choose model.

| You ask | The agent does |
|---|---|
| `"why is checkout-service slow right now?"` | Finds slow traces, identifies bottleneck spans, surfaces the root cause |
| `"any errors in the last hour?"` | Scans all services for errors, groups by service and type, shows log table |
| `"what's wrong with payment-service?"` | Correlates error logs with traces, highlights anomalies and failure spikes |
| `"show me the slowest endpoints today"` | Queries trace durations, ranks endpoints, returns a results table |
| `"mostrami i log di auth-service degli ultimi 10 minuti"` | Fetches and displays the log table directly in the chat |

Available models: `gpt-3.5-turbo` (faster) · `gpt-4` (more accurate)

---

## 🐳 Production Deployment

The production stack uses prebuilt images from GHCR — no repository clone needed on the host. Just copy `docker-compose.prod.yml`.

**Images:**
- `ghcr.io/evoluzione/opendashly:latest`
- `ghcr.io/evoluzione/opendashly-clickhouse:latest`
- `ghcr.io/evoluzione/opendashly-otel-collector:latest`

### Host requirements

| | Minimum | Recommended |
|---|---|---|
| CPU | 1 vCPU | 4 vCPU |
| RAM | 2 GB | 8 GB |
| Storage | 20 GB SSD | 50 GB+ SSD |

### Deploy

**1. Pick a VM profile.** Copy the matching `.env` block below next to `docker-compose.prod.yml` and fill in the three secrets on top.

| Profile | VM size | Use when |
|---|---|---|
| Small | 1 vCPU / 2 GB RAM | Very low traffic, test or small internal setup |
| Standard | 2 vCPU / 4 GB RAM | Small production baseline |
| Big | 4 vCPU / 8 GB RAM | Higher telemetry throughput |

Each block is a complete drop-in `.env` — no extra defaults to merge. All three assume the new resilience layer (`quantileTDigest`, per-query `SETTINGS`, halved-window retry, never-5xx widgets) so the dashboard keeps serving even when ClickHouse is memory-pressured.

<details>
<summary><b>Small — 1 vCPU / 2 GB</b></summary>

```bash
# --- Secrets (fill in) -------------------------------------------------------
AUTH_SECRET=replace-with-32-plus-random-chars
CLICKHOUSE_PASSWORD=replace-with-strong-password
CORS_ALLOWED_ORIGINS=http://your-host:5173

# --- Backend & dashboard -----------------------------------------------------
SERVICE_LIST_TIMEOUT_SECONDS=25
CLEANUP_INTERVAL_MINUTES=1440
RETENTION_COUNT_PRECHECK_ENABLED=false

CLICKHOUSE_MAX_MEMORY_MIB=96
CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_GROUP_BY_MIB=24
CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_SORT_MIB=24
CLICKHOUSE_MAX_TEMP_DATA_ON_DISK_MIB=512
CLICKHOUSE_MAX_EXECUTION_TIME_SECONDS=25
CLICKHOUSE_MAX_OPEN_CONNS=2
CLICKHOUSE_MAX_IDLE_CONNS=1
CLICKHOUSE_DIAL_TIMEOUT_SECONDS=5
CLICKHOUSE_READ_TIMEOUT_SECONDS=40

DASHBOARD_FRESH_CACHE_TTL_SECONDS=30
DASHBOARD_STALE_CACHE_TTL_SECONDS=900
DASHBOARD_REQUEST_TIMEOUT_SECONDS=25
DASHBOARD_QUERY_PARALLELISM=1
DASHBOARD_HALVE_ON_OOM=true

# --- OpenTelemetry Collector -------------------------------------------------
OTEL_FILE_STORAGE_DIR=/var/lib/otelcol/queue
OTEL_MEMORY_LIMITER_CHECK_INTERVAL=1s
OTEL_MEMORY_LIMIT_MIB=96
OTEL_MEMORY_SPIKE_LIMIT_MIB=20
OTEL_BATCH_SEND_SIZE=500
OTEL_BATCH_TIMEOUT=2s
OTEL_EXPORTER_TIMEOUT=10s
OTEL_SENDING_QUEUE_SIZE=5000
OTEL_SENDING_QUEUE_CONSUMERS=1
OTEL_RETRY_INITIAL_INTERVAL=1s
OTEL_RETRY_MAX_INTERVAL=30s
OTEL_RETRY_MAX_ELAPSED_TIME=0

# --- Container memory caps ---------------------------------------------------
APP_MEM_LIMIT=320m
APP_MEMSWAP_LIMIT=320m
OTEL_COLLECTOR_MEM_LIMIT=192m
OTEL_COLLECTOR_MEMSWAP_LIMIT=192m
CLICKHOUSE_MEM_LIMIT=896m
CLICKHOUSE_MEMSWAP_LIMIT=896m

# --- ClickHouse port bindings (localhost only on prod compose) ---------------
CLICKHOUSE_HTTP_PORT=8123
CLICKHOUSE_TCP_PORT=9000
```

</details>

<details>
<summary><b>Standard — 2 vCPU / 4 GB</b></summary>

```bash
# --- Secrets (fill in) -------------------------------------------------------
AUTH_SECRET=replace-with-32-plus-random-chars
CLICKHOUSE_PASSWORD=replace-with-strong-password
CORS_ALLOWED_ORIGINS=http://your-host:5173

# --- Backend & dashboard -----------------------------------------------------
SERVICE_LIST_TIMEOUT_SECONDS=20
CLEANUP_INTERVAL_MINUTES=720
RETENTION_COUNT_PRECHECK_ENABLED=false

CLICKHOUSE_MAX_MEMORY_MIB=160
CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_GROUP_BY_MIB=48
CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_SORT_MIB=48
CLICKHOUSE_MAX_TEMP_DATA_ON_DISK_MIB=768
CLICKHOUSE_MAX_EXECUTION_TIME_SECONDS=20
CLICKHOUSE_MAX_OPEN_CONNS=4
CLICKHOUSE_MAX_IDLE_CONNS=2
CLICKHOUSE_DIAL_TIMEOUT_SECONDS=5
CLICKHOUSE_READ_TIMEOUT_SECONDS=30

DASHBOARD_FRESH_CACHE_TTL_SECONDS=30
DASHBOARD_STALE_CACHE_TTL_SECONDS=900
DASHBOARD_REQUEST_TIMEOUT_SECONDS=20
DASHBOARD_QUERY_PARALLELISM=1
DASHBOARD_HALVE_ON_OOM=true

# --- OpenTelemetry Collector -------------------------------------------------
OTEL_FILE_STORAGE_DIR=/var/lib/otelcol/queue
OTEL_MEMORY_LIMITER_CHECK_INTERVAL=1s
OTEL_MEMORY_LIMIT_MIB=170
OTEL_MEMORY_SPIKE_LIMIT_MIB=35
OTEL_BATCH_SEND_SIZE=500
OTEL_BATCH_TIMEOUT=2s
OTEL_EXPORTER_TIMEOUT=10s
OTEL_SENDING_QUEUE_SIZE=5000
OTEL_SENDING_QUEUE_CONSUMERS=1
OTEL_RETRY_INITIAL_INTERVAL=1s
OTEL_RETRY_MAX_INTERVAL=30s
OTEL_RETRY_MAX_ELAPSED_TIME=0

# --- Container memory caps ---------------------------------------------------
APP_MEM_LIMIT=512m
APP_MEMSWAP_LIMIT=512m
OTEL_COLLECTOR_MEM_LIMIT=320m
OTEL_COLLECTOR_MEMSWAP_LIMIT=320m
CLICKHOUSE_MEM_LIMIT=2048m
CLICKHOUSE_MEMSWAP_LIMIT=2048m

# --- ClickHouse port bindings (localhost only on prod compose) ---------------
CLICKHOUSE_HTTP_PORT=8123
CLICKHOUSE_TCP_PORT=9000
```

</details>

<details>
<summary><b>Big — 4 vCPU / 8 GB</b></summary>

```bash
# --- Secrets (fill in) -------------------------------------------------------
AUTH_SECRET=replace-with-32-plus-random-chars
CLICKHOUSE_PASSWORD=replace-with-strong-password
CORS_ALLOWED_ORIGINS=http://your-host:5173

# --- Backend & dashboard -----------------------------------------------------
SERVICE_LIST_TIMEOUT_SECONDS=15
CLEANUP_INTERVAL_MINUTES=360
RETENTION_COUNT_PRECHECK_ENABLED=false

CLICKHOUSE_MAX_MEMORY_MIB=256
CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_GROUP_BY_MIB=64
CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_SORT_MIB=64
CLICKHOUSE_MAX_TEMP_DATA_ON_DISK_MIB=1024
CLICKHOUSE_MAX_EXECUTION_TIME_SECONDS=15
CLICKHOUSE_MAX_OPEN_CONNS=8
CLICKHOUSE_MAX_IDLE_CONNS=4
CLICKHOUSE_DIAL_TIMEOUT_SECONDS=5
CLICKHOUSE_READ_TIMEOUT_SECONDS=20

DASHBOARD_FRESH_CACHE_TTL_SECONDS=30
DASHBOARD_STALE_CACHE_TTL_SECONDS=900
DASHBOARD_REQUEST_TIMEOUT_SECONDS=15
DASHBOARD_QUERY_PARALLELISM=2
DASHBOARD_HALVE_ON_OOM=true

# --- OpenTelemetry Collector -------------------------------------------------
OTEL_FILE_STORAGE_DIR=/var/lib/otelcol/queue
OTEL_MEMORY_LIMITER_CHECK_INTERVAL=1s
OTEL_MEMORY_LIMIT_MIB=300
OTEL_MEMORY_SPIKE_LIMIT_MIB=60
OTEL_BATCH_SEND_SIZE=500
OTEL_BATCH_TIMEOUT=2s
OTEL_EXPORTER_TIMEOUT=10s
OTEL_SENDING_QUEUE_SIZE=5000
OTEL_SENDING_QUEUE_CONSUMERS=1
OTEL_RETRY_INITIAL_INTERVAL=1s
OTEL_RETRY_MAX_INTERVAL=30s
OTEL_RETRY_MAX_ELAPSED_TIME=0

# --- Container memory caps ---------------------------------------------------
APP_MEM_LIMIT=768m
APP_MEMSWAP_LIMIT=768m
OTEL_COLLECTOR_MEM_LIMIT=512m
OTEL_COLLECTOR_MEMSWAP_LIMIT=512m
CLICKHOUSE_MEM_LIMIT=4096m
CLICKHOUSE_MEMSWAP_LIMIT=4096m

# --- ClickHouse port bindings (localhost only on prod compose) ---------------
CLICKHOUSE_HTTP_PORT=8123
CLICKHOUSE_TCP_PORT=9000
```

</details>

<br/>

**2. Resilience behavior** — shared across all profiles:

- ad-hoc query execution returns `status: "partial"` with per-signal failures in `signalErrors`
- dashboard metrics always return a response: any failing widget (recoverable or not) is reported via `warnings`, never a 5xx
- on a `memory limit exceeded` error each widget retries once over the last half of the requested window (warning: `partial window (last half) due to backend pressure`)
- dashboard queries use `quantileTDigest` and inline `SETTINGS` so a single hot widget can't exhaust ClickHouse on small VMs
- `otel-collector` uses persistent queue storage (`file_storage`) and `blocking: true`, so under pressure it applies backpressure instead of dropping data aggressively. If backlog stays high for a long period, tune queue size first, then collector/ClickHouse resources.

**3. Start:**

```bash
docker compose -f docker-compose.prod.yml up -d
```

**4. Verify:**

```bash
curl http://localhost:8080/healthz   # → backend
curl http://localhost:8123/ping      # → ClickHouse
curl http://localhost:13133          # → OTel Collector health
```

Then open `http://localhost:5173` and log in with `admin` / `admin`.

### Security checklist

- [ ] Change default admin password immediately
- [ ] Set a strong `AUTH_SECRET` (32+ random characters)
- [ ] Enable HTTPS/TLS for all public-facing services
- [ ] Block direct ClickHouse ports (8123, 9000) from the internet
- [ ] Restrict `CORS_ALLOWED_ORIGINS` to your actual domains
- [ ] Enable API rate limiting
- [ ] Configure data retention policies
- [ ] Set up ClickHouse backups

---

## 🙏 Built on open source

[OpenTelemetry](https://opentelemetry.io) · [ClickHouse](https://clickhouse.com) · [Go](https://golang.org) · [SvelteKit](https://kit.svelte.dev) · [uPlot](https://github.com/leeoniya/uPlot) · [Chi](https://github.com/go-chi/chi)
