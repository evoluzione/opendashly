<div align="center">

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
- **Automatic diagnosis** — describe a service or time window in plain language; it compares against the previous window and surfaces anomalies, error origins, and evidence, with no LLM or API key
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

| Service     | URL                   |
| ----------- | --------------------- |
| Dashboard   | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| OTLP HTTP   | http://localhost:4318 |
| OTLP gRPC   | grpc://localhost:4317 |
| ClickHouse  | http://localhost:8123 |

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

| Flag             | Description                              | Default                   |
| ---------------- | ---------------------------------------- | ------------------------- |
| `--duration`   | Test duration in seconds                 | —                        |
| `--rps`        | Target requests per second               | —                        |
| `--services`   | Active microservices (2–9)              | 9                         |
| `--error-rate` | Base error probability (0–1)            | 0.02                      |
| `--hot-rate`   | Extra traffic on checkout hotspot (0–1) | 0.2                       |
| `--no-metrics` | Disable metrics signal                   | —                        |
| `--no-traces`  | Disable traces signal                    | —                        |
| `--collector`  | OTLP HTTP base URL                       | `http://localhost:4318` |

</details>

---

## 🤖 Automatic Diagnosis

Opendashly ships a built-in diagnosis assistant that runs entirely inside the backend: no LLM, no API key, no external service, and it fits the minimal 1 CPU / 2 GB deployment. It understands Italian and English, sloppy typing and typos.

- **It asks before it analyzes.** If the time window or the service is missing, it restates what it understood and asks, with one-click answers.
- **Short answers first.** One sentence with the verdict and at most three numbered points; ask for `dettagli` / `details` for the full report.
- **It remembers the answer it just gave.** `analizza il primo`, `the second one`, `quello del carrello`, `e ieri?`, `and latency?` refer to the previous answer.
- **Precise numbers.** `latenza media delle GET del catalogo`, `how many errors did payment have today`, `p95 di /checkout`.
- **Honest limits.** Charts, restarts, alert setup, exports and business metrics are declined instead of answered with unrelated data.

| You ask                                                     | The assistant                                                                                              |
| ----------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `"ci sono errori?"`                                         | asks the window, then the service, then checks error rate, hotspots, error logs and where errors originate |
| `"checkout-service ultime 2 ore"`                           | compares with the previous 2 hours: error rate, traffic, p95 regressions, new error hotspots, log patterns |
| `"voglio sapere in media la velocità delle GET del catalogo"` | asks the window, then gives the request-weighted average latency and the slowest GET endpoint             |
| `"4bf92f3577b34da6a3ce929d0e0e4736"`                        | finds the deepest errored span (where the failure started), the slowest span and linked error logs         |

The rules are checked against labeled IT/EN corpora in `backend/internal/application/diagnosis/testdata/`.

---

## 🐳 Production Deployment

The production stack uses prebuilt images from GHCR — no repository clone needed on the host. Just copy `docker-compose.prod.yml`.

**Images:**

- `ghcr.io/evoluzione/opendashly:latest`
- `ghcr.io/evoluzione/opendashly-clickhouse:latest`
- `ghcr.io/evoluzione/opendashly-otel-collector:latest`

### Host requirements

|         | Minimum   | Recommended |
| ------- | --------- | ----------- |
| CPU     | 1 vCPU    | 4 vCPU      |
| RAM     | 2 GB      | 8 GB        |
| Storage | 20 GB SSD | 50 GB+ SSD  |

### Deploy

**1. Set only secrets and CORS.** Create a `.env` next to `docker-compose.prod.yml`. There are no machine profiles and no runtime tuning variables to set.

```bash
AUTH_SECRET=replace-with-32-plus-random-chars
CLICKHOUSE_PASSWORD=replace-with-strong-password
CORS_ALLOWED_ORIGINS=http://your-host:5173
```

**Self-tuning instead of profiles.** The old `small`/`standard`/`big` profiles are gone. Each component adapts on its own and stays as resilient as possible:

- **Backend** — auto-sizes fixed operational defaults from cgroup CPU/RAM, starts load-sensitive knobs (dashboard query parallelism, telemetry concurrency, per-query ClickHouse memory) at a safe floor, then adapts them at runtime with an AIMD controller: it grows them by one step after sustained calm and cuts them sharply the moment it detects pressure (memory, ClickHouse disk, or recoverable OOM/timeout errors).
- **OTel collector** — sizes its `memory_limiter`, batching and queue knobs at boot from the container cgroup limit, falling back to host RAM. There is no external override.
- **ClickHouse** — scales server memory from available RAM via `max_server_memory_usage_to_ram_ratio`; per-query budgets come from the backend's adaptive settings.

Heavy telemetry queries stay isolated so the UI remains navigable under pressure.

**2. Resilience behavior:**

- ad-hoc query execution returns `status: "partial"` with per-signal failures in `signalErrors`
- dashboard metrics always return a response: any failing widget (recoverable or not) is reported via `warnings`, never a 5xx
- dashboard and system monitor metrics read bounded minute rollups; raw telemetry fallback is disabled by default for operational screens
- auth/settings/status use a separate ClickHouse control lane while telemetry queries use a bounded telemetry lane
- on `memory limit exceeded` / `OvercommitTracker`, the dashboard opens a short pressure cooldown and serves the last good snapshot instead of retrying expensive widgets
- dashboard rollups use `quantilesTDigest` and inline `SETTINGS` so a single hot widget can't exhaust ClickHouse on small VMs
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
- [ ] Review data retention policies
- [ ] Set up ClickHouse backups

---

## 🙏 Built on open source

[OpenTelemetry](https://opentelemetry.io) · [ClickHouse](https://clickhouse.com) · [Go](https://golang.org) · [SvelteKit](https://kit.svelte.dev) · [uPlot](https://github.com/leeoniya/uPlot) · [Chi](https://github.com/go-chi/chi)
