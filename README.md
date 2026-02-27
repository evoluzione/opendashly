<div align="center">

# 📊 Opendashly

### Complete Observability Platform with Logs, Traces, and Metrics

[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.20-4F46E5?style=flat&logo=opentelemetry)](https://opentelemetry.io)
[![ClickHouse](https://img.shields.io/badge/ClickHouse-24-FFCC01?style=flat&logo=clickhouse)](https://clickhouse.com)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-4.2-FF3E00?style=flat&logo=svelte)](https://kit.svelte.dev)
[![OpenAI](https://img.shields.io/badge/OpenAI-GPT--3.5/4-412991?style=flat&logo=openai)](https://openai.com)

 [Quick Start](#-quick-start) • [AI Features](#ai) • [Architecture](#architecture) • [Production Deployment](#-docker-deployment) • [Monitoring](#-monitoring)

</div>

---

## 🎯 Overview

**Opendashly** is a production-ready observability platform that ingests, stores, and visualizes telemetry data from OpenTelemetry-instrumented applications. Built with performance and scalability in mind, it provides a unified interface for exploring logs, traces, and metrics.

### Why Opendashly?

- **🚀 High Performance**: ClickHouse-powered storage handles millions of events per second
- **🤖 AI Smart Queries**: Natural language to SQL query generation with OpenAI
- **🔐 Enterprise Auth**: JWT-based authentication with role-based access control
- **📈 Real-time Visualization**: Interactive charts and timelines with sub-second queries
- **🎛️ Data Retention**: Configurable retention policies with automatic cleanup
- **🔍 Trace Correlation**: Seamlessly navigate from logs to traces and spans
- **🐳 Docker Ready**: Full stack deployable with a single command

---

## 🚀 Quick Start

### Prerequisites

- **Docker** 20.10+ and **Docker Compose** 2.0+
- (Optional) **Go** 1.22+ for local backend development
- (Optional) **Node.js** 18+ for local frontend development

### 1. Start All Services

If you want custom secrets or a non-default ClickHouse password, copy `.env.example` to `.env` and edit it first.

```bash
docker compose up --build
```

Telemetry schema is now managed by backend migrations (`backend/internal/storage/migrations/000_otel_schema_baseline.sql`).
The collector is configured with `create_schema: false` and only writes data.

For a **fresh bootstrap** after this change (or if you want to rebuild from zero), reset volumes once:

```bash
docker compose down -v
docker compose up --build
```

This starts:
- **Frontend** → [http://localhost:5173](http://localhost:5173)
- **Backend API** → [http://localhost:8080](http://localhost:8080)
- **ClickHouse** → [http://localhost:8123](http://localhost:8123) (HTTP), `tcp://localhost:9000`
- **OTLP Collector** → `grpc://localhost:4317`, `http://localhost:4318`

### 2. Login

Navigate to [http://localhost:5173](http://localhost:5173) and login with:

```
Username: admin
Password: admin
```

⚠️ **You will be prompted to change the password on first login.**

### 3. Send Test Data

Send OpenTelemetry data to the collector:

```bash
# HTTP endpoint
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "test-service"}
        }]
      },
      "scopeLogs": [{
        "logRecords": [{
          "timeUnixNano": "'$(date +%s)'000000000",
          "severityText": "INFO",
          "body": {"stringValue": "Hello from OpenTelemetry!"}
        }]
      }]
    }]
  }'
```

Or configure your application to send telemetry to `http://localhost:4318` (HTTP) or `grpc://localhost:4317` (gRPC).

---

<a id="ai"></a>
## 🤖 AI-Powered Query Generation

Opendashly integrates with **OpenAI** to enable natural language query generation for exploring your observability data.

### Features

- **Natural Language to SQL**: Describe what you want in plain language (Italian or English), and the AI generates optimized ClickHouse SQL queries
- **Schema-Aware**: The AI understands your telemetry schema (logs, metrics, traces) and generates appropriate queries
- **Context Detection**: Automatically detects whether you're asking about logs, metrics, or traces
- **Secure Storage**: API keys are encrypted and stored per-tenant

### Configuration

1. **Via Settings UI**: Navigate to Settings → AI Configuration in the dashboard
2. **Enable AI**: Toggle the AI feature on
3. **Enter API Key**: Provide your OpenAI API key
4. **Select Model**: Choose between `gpt-3.5-turbo` (default, faster) or `gpt-4` (more accurate)

### Usage Examples

In the query form, type natural language prompts like:

| Prompt | Generated Query |
|--------|-----------------|
| `"mostrami i log degli ultimi 5 minuti"` | Logs from last 5 minutes |
| `"errori del servizio auth-service nell'ultima ora"` | Error logs for auth-service, last hour |
| `"tracce più lente degli ultimi 15 minuti"` | Slowest traces from last 15 minutes |
| `"metriche CPU per il servizio api"` | CPU metrics for api service |

---

<a id="architecture"></a>
## 🏗️ Architecture

### Data Flow

```
┌─────────────┐      ┌──────────────────┐      ┌─────────────┐
│   SDK/Agent │─────▶│ OTLP Collector   │─────▶│ ClickHouse  │
│  (Your App) │      │  (4317/4318)     │      │   Storage   │
└─────────────┘      └──────────────────┘      └─────────────┘
                                                       │
                                                       ▼
                     ┌──────────────────┐      ┌─────────────┐
                     │   SvelteKit UI   │◀─────│  Go API     │
                     │  (Frontend)      │      │  (Backend)  │
                     └──────────────────┘      └─────────────┘
```

### Tech Stack

#### Backend
- **Language**: Go 1.22+
- **Framework**: Chi Router (lightweight, idiomatic HTTP framework)
- **Database**: ClickHouse 24+ (columnar OLAP database)
- **Auth**: JWT with bcrypt password hashing
- **Driver**: `clickhouse-go/v2` (native protocol)

#### Frontend
- **Framework**: SvelteKit 4.2+ (SSR-capable framework)
- **Charts**: uPlot 1.6+ (high-performance time-series charts)
- **Build Tool**: Vite 5.0+
- **Testing**: Vitest (unit), Playwright (E2E)

#### Infrastructure
- **Telemetry**: OpenTelemetry Collector Contrib 0.122.0
- **Protocol**: OTLP (OpenTelemetry Protocol)
- **Containerization**: Docker + Docker Compose

### Directory Structure

```
.
├── backend/                  # Go API server
│   ├── cmd/api/             # Application entry point
│   ├── internal/
│   │   ├── api/             # HTTP handlers and routing
│   │   ├── auth/            # Authentication & authorization
│   │   ├── query/           # Query execution and builders
│   │   ├── retention/       # Data retention & cleanup
│   │   ├── storage/         # ClickHouse client & migrations
│   │   └── config/          # Configuration management
│   └── tests/               # Integration & contract tests
├── frontend/                 # SvelteKit application
│   ├── src/
│   │   ├── routes/          # SvelteKit pages
│   │   ├── components/      # Reusable Svelte components
│   │   ├── lib/stores/      # State management
│   │   └── services/        # API client functions
│   └── tests/               # Unit & E2E tests
├── collector-config.yaml     # OTLP Collector configuration
├── docker-compose.yml        # Multi-container orchestration
├── docker-compose.prod.yml   # Production compose (single app image)
├── Dockerfile                # Combined backend+frontend image build
├── docker/                   # Runtime helpers
│   └── entrypoint.sh         # Starts backend + frontend
├── .env.example              # Environment variable template
├── .github/workflows/        # CI pipelines
│   └── ci-docker.yml         # Tests + image build/push
└── README.md                 # You are here
```

---

## 🚀 Docker Deployment

### Prerequisites

- Docker 20.10+ and Docker Compose 2.0+
- Host sizing (single node): 2 vCPU and 4 GB RAM minimum; 4 vCPU and 8 GB RAM recommended for moderate workloads
- Storage: SSD recommended; allocate at least 20 GB free space for ClickHouse data
- A strong `AUTH_SECRET` value (32+ random characters)
- A secure `CLICKHOUSE_PASSWORD`

### Setup Guide

1) Copy `docker-compose.prod.yml` and `collector-config.yaml` to the target host.

2) Create a `.env` file alongside `docker-compose.prod.yml`:

```bash
AUTH_SECRET=replace-with-32+char-random
CLICKHOUSE_PASSWORD=replace-with-strong-password
CORS_ALLOWED_ORIGINS=http://your-public-host:5173
```

3) Start the stack:

```bash
docker compose -f docker-compose.prod.yml up -d
```

4) Verify services:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8123/ping
```

5) Open the UI and log in:

- Frontend: http://localhost:5173
- Default credentials: `admin` / `admin`
- You will be prompted to change the password on first login.

### Security Checklist

- [ ] Change default admin password immediately
- [ ] Set strong `AUTH_SECRET` (32+ random characters)
- [ ] Use HTTPS/TLS for all services
- [ ] Enable ClickHouse authentication
- [ ] Configure firewall rules (block direct ClickHouse access)
- [ ] Review and restrict CORS settings in `router.go`
- [ ] Enable rate limiting for API endpoints
- [ ] Set up log retention policies
- [ ] Configure backup strategy for ClickHouse

---

## 📊 Monitoring

### Health Checks

```bash
# Backend health
curl http://localhost:8080/healthz

# ClickHouse health
curl http://localhost:8123/ping

# Collector health (metrics endpoint)
curl http://localhost:8888/metrics
```

### Metrics

The dashboard exposes Prometheus-compatible metrics at `/metrics` (if enabled):
- Request latency histograms
- Query execution duration
- Active connections
- Error rates


---

## 🙏 Acknowledgments

Built with these excellent open-source projects:

- [OpenTelemetry](https://opentelemetry.io) - Observability framework
- [ClickHouse](https://clickhouse.com) - High-performance columnar database
- [Go](https://golang.org) - Backend language
- [SvelteKit](https://kit.svelte.dev) - Frontend framework
- [uPlot](https://github.com/leeoniya/uPlot) - High-performance charting
- [Chi](https://github.com/go-chi/chi) - Lightweight Go router

---

<div align="center">

**[⬆ Back to Top](#-opendashly)**

</div>
