# Quickstart: OpenTelemetry Telemetry Dashboard

## Prerequisites
- Docker and Docker Compose
- Go 1.22
- Node.js 20 LTS

## 1) Start ClickHouse
```bash
docker run -d --name clickhouse \
  -p 8123:8123 -p 9000:9000 \
  clickhouse/clickhouse-server:24
```

## 2) Start OpenTelemetry Collector
Create `collector-config.yaml`:
```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:
exporters:
  clickhouse:
    endpoint: tcp://localhost:9000
    database: telemetry
    timeout: 5s
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

Run the collector:
```bash
docker run -d --name otel-collector \
  -p 4317:4317 -p 4318:4318 \
  -v $(pwd)/collector-config.yaml:/etc/otelcol/config.yaml \
  otel/opentelemetry-collector-contrib:0.99.0
```

## 3) Run the Go API
```bash
cd backend
export CLICKHOUSE_ADDR="localhost:9000"
export API_LISTEN_ADDR=":8080"
export AUTH_MODE="header"
go run ./cmd/api
```

## 4) Run the SvelteKit UI
```bash
cd frontend
npm install
npm run dev
```

## 5) Verify
- Open `http://localhost:5173`
- Run a query in the UI to see logs, traces, and metrics
