# OpenTelemetry Telemetry Dashboard

## Avvio rapido (Docker)

```bash
docker compose -f deployments/docker-compose.yml up --build
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
- `VITE_API_BASE` (es. `http://backend:8080` quando in docker)

## Note
- L'autenticazione demo usa header `X-Tenant-ID` e `X-User-ID`.
- Il Collector usa `deployments/collector-config.yaml` con export su ClickHouse.
