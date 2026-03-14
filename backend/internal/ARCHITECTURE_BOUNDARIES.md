# Backend Architecture Boundaries (Phase A)

## Goal

Define and freeze package boundaries before moving directories.
This phase is documentation-first: no behavior changes, only an explicit dependency map and import rules.

## Current Package Inventory

- opendashly/backend/internal/ai
- opendashly/backend/internal/api
- opendashly/backend/internal/api/handlers
- opendashly/backend/internal/auth
- opendashly/backend/internal/bootstrap
- opendashly/backend/internal/config
- opendashly/backend/internal/dashboard
- opendashly/backend/internal/metrics
- opendashly/backend/internal/query
- opendashly/backend/internal/query/builders
- opendashly/backend/internal/retention
- opendashly/backend/internal/status
- opendashly/backend/internal/storage
- opendashly/backend/internal/telemetry

## Current Internal Dependency Map (Baseline)

Source: `go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/...`

- `internal/ai` -> none
- `internal/api` -> `internal/ai`, `internal/api/handlers`, `internal/config`, `internal/dashboard`, `internal/metrics`, `internal/query`, `internal/status`
- `internal/api/handlers` -> `internal/ai`, `internal/auth`, `internal/dashboard`, `internal/metrics`, `internal/query`, `internal/retention`, `internal/status`
- `internal/auth` -> none
- `internal/bootstrap` -> `internal/ai`, `internal/api`, `internal/api/handlers`, `internal/auth`, `internal/config`, `internal/dashboard`, `internal/metrics`, `internal/query`, `internal/retention`, `internal/status`, `internal/storage`
- `internal/config` -> none
- `internal/dashboard` -> none
- `internal/metrics` -> `internal/storage`
- `internal/query` -> `internal/ai`, `internal/query/builders`, `internal/storage`
- `internal/query/builders` -> none
- `internal/retention` -> none
- `internal/status` -> `internal/storage`
- `internal/storage` -> none
- `internal/telemetry` -> none

## Target Layers (for directory reordering)

- `interfaces/http`: transport layer (router, handlers, request/response DTOs)
- `application`: use-case orchestration (query run, metrics/dashboard composition, auth flows)
- `domain`: entities/value objects/pure rules
- `infrastructure`: ClickHouse, SQL builders, repositories, migrations
- `shared`: cross-cutting helpers and utilities

## Import Rules (to enforce in next phases)

1. `interfaces/http` can import only `application` and `shared`.
2. `application` can import `domain`, `infrastructure` abstractions, and `shared`.
3. `domain` cannot import `interfaces/http` or concrete DB packages.
4. `infrastructure` cannot import `interfaces/http`.
5. `bootstrap` is the only composition root and can wire all layers.

## Package Classification (Phase A proposal)

- `internal/api`, `internal/api/handlers` -> future `interfaces/http`
- `internal/query` -> split between `application/query` and `domain/query`
- `internal/query/builders`, `internal/storage` -> future `infrastructure/*`
- `internal/metrics`, `internal/dashboard`, `internal/status` -> mostly `application/*` (+ `infrastructure` query adapters)
- `internal/auth`, `internal/retention`, `internal/ai` -> split by use-case/domain/infrastructure in later phases
- `internal/bootstrap` -> keep as composition root

## Hotspots To Address During Moves

- `internal/bootstrap` currently imports many concrete modules.
- `internal/api` depends directly on concrete services.
- `internal/query` still mixes application orchestration and infra-facing concerns.
- `internal/metrics` still contains orchestration and query concerns in the same package.

## Acceptance Criteria For Phase A

1. Architecture boundary document exists in repository.
2. Baseline internal dependency map is captured.
3. Import rules are explicit and reviewed.
4. Target package classification is agreed before file moves.

## Regeneration Command

Run from `backend/`:

`go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/...`
