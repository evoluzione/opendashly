# OpenTelemetry Telemetry Dashboard

Dashboard per osservabilita` basata su OpenTelemetry: raccoglie tracce/metriche/log, le salva in ClickHouse e le visualizza con una UI web.

## Come funziona

Flusso dati:
- SDK/Agent -> OTLP (gRPC 4317 / HTTP 4318)
- OpenTelemetry Collector -> ClickHouse
- API Go (chi) -> query su ClickHouse
- Frontend SvelteKit -> API per visualizzazione

Componenti principali:
- `collector-config.yaml` definisce pipeline OTLP -> ClickHouse.
- `backend/` espone API di lettura e auth demo.
- `frontend/` mostra dashboard e grafici (uPlot).

## Avvio rapido (Docker)

```bash
docker compose up --build
```

Servizi esposti:
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- ClickHouse: http://localhost:8123 (HTTP), tcp://localhost:9000
- OTLP: grpc 4317, http 4318

## Variabili ambiente

Backend:
- `CLICKHOUSE_ADDR` (default richiesto, es. `clickhouse:9000`)
- `API_LISTEN_ADDR` (default `:8080`)
- `AUTH_MODE` (default `header`)

Frontend:
- `VITE_API_BASE` (es. `http://localhost:8080` dal browser)

## Note
- L'autenticazione demo usa header `X-Tenant-ID` e `X-User-ID`.
- Il Collector usa `collector-config.yaml` con export su ClickHouse.
