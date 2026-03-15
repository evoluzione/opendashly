# Backend Architecture Boundaries

## Goal

Keep backend internals organized under 4 top-level layers:

- `internal/interfaces`
- `internal/application`
- `internal/domain`
- `internal/infrastructure`

## Current Package Inventory

- `opendashly/backend/internal/interfaces/api`
- `opendashly/backend/internal/interfaces/http`
- `opendashly/backend/internal/interfaces/http/handlers`
- `opendashly/backend/internal/application/ai`
- `opendashly/backend/internal/application/auth`
- `opendashly/backend/internal/application/bootstrap`
- `opendashly/backend/internal/application/dashboard`
- `opendashly/backend/internal/application/metrics`
- `opendashly/backend/internal/application/query`
- `opendashly/backend/internal/application/retention`
- `opendashly/backend/internal/application/status`
- `opendashly/backend/internal/domain/telemetry`
- `opendashly/backend/internal/infrastructure/config`
- `opendashly/backend/internal/infrastructure/querysql`
- `opendashly/backend/internal/infrastructure/querysql/builders`
- `opendashly/backend/internal/infrastructure/storage`

## Layer Responsibilities

- `interfaces`: transport adapters (HTTP router, handlers, DTO mapping).
- `application`: use case orchestration and service composition.
- `domain`: domain models and pure business concepts.
- `infrastructure`: external systems (ClickHouse, migrations, SQL builders, runtime config).

## Import Rules

1. `interfaces` can import `application` and `domain` DTO/value types when needed.
2. `application` can import `domain` and `infrastructure` packages.
3. `domain` must not import `interfaces` or concrete infrastructure packages.
4. `infrastructure` must not import `interfaces`.
5. Composition root lives in `application/bootstrap` and can wire all layers.

## Conventions

1. New backend packages must be created under one of the 4 top-level directories.
2. Legacy paths under `internal/<old-package>` must not be reintroduced.
3. If a package does not clearly fit, prefer `application` first and split to `domain` only when invariants become explicit.

## Regeneration Command

Run from `backend/`:

`go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/...`
