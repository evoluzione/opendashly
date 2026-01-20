<div align="center">

# 📊 Opendashly

### Complete Observability Platform with Logs, Traces, and Metrics

[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.20-4F46E5?style=flat&logo=opentelemetry)](https://opentelemetry.io)
[![ClickHouse](https://img.shields.io/badge/ClickHouse-24-FFCC01?style=flat&logo=clickhouse)](https://clickhouse.com)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-4.2-FF3E00?style=flat&logo=svelte)](https://kit.svelte.dev)

[Features](#-features) • [Quick Start](#-quick-start) • [Architecture](#-architecture)

</div>

---

## 🎯 Overview

**Opendashly** is a production-ready observability platform that ingests, stores, and visualizes telemetry data from OpenTelemetry-instrumented applications. Built with performance and scalability in mind, it provides a unified interface for exploring logs, traces, and metrics.

### Why Opendashly?

- **🚀 High Performance**: ClickHouse-powered storage handles millions of events per second
- **🔐 Enterprise Auth**: JWT-based authentication with role-based access control
- **📈 Real-time Visualization**: Interactive charts and timelines with sub-second queries
- **🎛️ Data Retention**: Configurable retention policies with automatic cleanup
- **🔍 Trace Correlation**: Seamlessly navigate from logs to traces and spans
- **🐳 Docker Ready**: Full stack deployable with a single command

---

## ✨ Features

### Core Capabilities

#### 📝 **Logs Management**
- Full-text search with filter syntax support
- Severity-level filtering (DEBUG, INFO, WARN, ERROR, FATAL)
- Service and attribute-based filtering
- Trace ID correlation for distributed tracing
- Real-time log streaming

#### 🔗 **Distributed Tracing**
- Trace timeline visualization with span relationships
- Service dependency mapping
- Duration analysis and latency tracking
- Span attribute inspection
- Error and status tracking

#### 📊 **Metrics Visualization**
- Time-series charts with uPlot (high-performance rendering)
- Gauge and counter metric support
- Custom time range selection
- Multi-metric comparison
- Service-level aggregations

### Advanced Features

#### 🛡️ **Authentication & Authorization**
- JWT-based session management
- Role-based access control (Admin, User)
- Secure password hashing with bcrypt
- Mandatory password change on first login
- Session persistence with HTTP-only cookies

#### 🗄️ **Data Retention & Cleanup**
- Configurable retention periods per signal type (logs, traces, metrics)
- Automatic background cleanup (default: 24-hour interval)
- Manual cleanup with service filtering
- Cleanup job audit trail
- Admin-only retention management

#### 💾 **Query Management**
- Save and reuse complex queries
- Shareable query templates
- Query history tracking
- Multi-signal queries (logs + traces + metrics)

#### 🔎 **Search Modes**
- **Automatic**: pick a quick range and refresh results periodically
- **Manual**: set start/end date and time explicitly
- **Smart**: write a natural-language prompt and generate the query automatically

---

## 🚀 Quick Start

### Prerequisites

- **Docker** 20.10+ and **Docker Compose** 2.0+
- (Optional) **Go** 1.22+ for local backend development
- (Optional) **Node.js** 18+ for local frontend development

### 1. Start All Services

```bash
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
- **Telemetry**: OpenTelemetry Collector Contrib 0.99.0
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
└── README.md                 # You are here
```

---

## 🔧 Configuration

### Collector Configuration

The OTLP Collector is configured via `collector-config.yaml`. Key sections:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  clickhouse:
    endpoint: tcp://clickhouse:9000
    database: telemetry
    ttl_days: 7  # Default retention
    timeout: 10s

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [clickhouse]
    metrics:
      receivers: [otlp]
      exporters: [clickhouse]
    logs:
      receivers: [otlp]
      exporters: [clickhouse]
```

### Data Retention

Configure retention policies via the admin UI (`/admin/retention`)

**Retention Features:**
- Default 7-day retention for all signal types
- Configurable per signal type (logs, traces, metrics)
- Automatic cleanup every 24 hours (configurable)
- Manual cleanup with service filtering
- Audit trail for all cleanup operations

---

## 🚀 Production Deployment

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

### Recommended Production Setup

```yaml
# docker-compose.prod.yml
services:
  backend:
    environment:
      - AUTH_SECRET=${AUTH_SECRET}  # From .env file
      - CLICKHOUSE_ADDR=clickhouse:9000
      - AUTH_MODE=jwt
      - CLEANUP_INTERVAL_MINUTES=1440
    restart: unless-stopped

  clickhouse:
    environment:
      - CLICKHOUSE_USER=telemetry
      - CLICKHOUSE_PASSWORD=${CLICKHOUSE_PASSWORD}
    volumes:
      - clickhouse_data:/var/lib/clickhouse
    restart: unless-stopped
```

### Performance Tuning

#### ClickHouse Optimization
- Adjust `max_memory_usage` based on available RAM
- Configure `max_threads` for query parallelization
- Use materialized views for common aggregations
- Enable compression for cold storage

#### Backend Optimization
- Increase connection pool size for high load
- Enable HTTP/2 for multiplexing
- Configure request timeouts appropriately
- Use caching for frequent queries

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

## 📧 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/opendashly/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/opendashly/discussions)

---

<div align="center">

**[⬆ Back to Top](#-opendashly)**

</div>
