# Implementation Plan: Secure Dashboard Access and UI Restyle

**Branch**: `001-dashboard-auth-restyle` | **Date**: 2026-01-14 | **Spec**: `specs/001-dashboard-auth-restyle/spec.md`
**Input**: Feature specification from `/specs/001-dashboard-auth-restyle/spec.md`

## Summary

Add authenticated access with a first-login password change for the default admin, provide admin user management, and restyle the dashboard with a left sidebar and consistent filters for Logs/Metriche/Tracce. Implementation updates the API auth model, adds user storage, and refactors frontend layout without altering telemetry query semantics.

## Technical Context

**Language/Version**: Go 1.22 (API), Node.js 20 LTS (SvelteKit)  
**Primary Dependencies**: chi router, SvelteKit, ClickHouse driver, OpenTelemetry Collector, uPlot  
**Storage**: ClickHouse (telemetry data + user/auth tables)  
**Testing**: go test (backend), npm test (frontend)  
**Target Platform**: Linux server + modern browsers  
**Project Type**: web  
**Performance Goals**: Auth endpoints complete in <300ms p95; UI tab switches and filter updates render in <2s for typical queries  
**Constraints**: No dashboard content without authentication; default admin must change password before access; filters remain visible and user-controlled  
**Scale/Scope**: Single-tenant deployments with small teams (tens of users)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- OTLP ingest path specified; schema preservation and any normalization documented: No changes to ingest; existing OTLP pipeline and schemas remain unchanged.
- Query model defined; saved queries and visibility of filters guaranteed: Queries stay explicit; all filters remain visible in the UI layout.
- Data fidelity plan defined (immutability, lineage, sampling/rollups): No changes to storage fidelity or rollup behavior.
- Performance budgets defined for ingest and query latency at target scale: Budgets documented in Technical Context; no ingest impact.
- Security model defined (authn/authz, tenant isolation, data protection): Add session-based auth with role-based admin access; keep single-tenant scope.

## Project Structure

### Documentation (this feature)

```text
specs/001-dashboard-auth-restyle/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── cmd/
├── internal/
│   ├── api/
│   ├── auth/
│   ├── config/
│   ├── query/
│   └── storage/
└── tests/

frontend/
├── src/
│   ├── lib/
│   ├── routes/
│   └── components/
└── tests/

tests/
```

**Structure Decision**: Web application with separate `backend/` and `frontend/` directories already present in the repository.

## Phase 0: Outline & Research

### Research Tasks

1. Confirm secure password storage approach for Go services with ClickHouse-backed user records.
2. Choose an authentication mechanism that works with the existing API gateway and SvelteKit frontend (session cookies vs token-based).
3. Identify best practices for first-login password change enforcement.
4. Identify UI layout patterns for sidebar navigation + filter rail to ensure accessibility and responsiveness.

### Output: `research.md`

See `specs/001-dashboard-auth-restyle/research.md`.

## Phase 1: Design & Contracts

### Data Model

Define the user and auth-related entities, including roles and first-login flags. See `specs/001-dashboard-auth-restyle/data-model.md`.

### API Contracts

Define authentication and user management endpoints plus service listing used by the UI. Contracts located in `specs/001-dashboard-auth-restyle/contracts/`.

### Quickstart

Document the minimal steps to log in, change the default password, and create users. See `specs/001-dashboard-auth-restyle/quickstart.md`.

### Agent Context Update

Run `.specify/scripts/powershell/update-agent-context.ps1 -AgentType codex` after `plan.md` is completed to sync `AGENTS.md`.

### Constitution Check (Post-Design)

- OTLP ingest path specified; schema preservation and any normalization documented: No changes required for this feature.
- Query model defined; saved queries and visibility of filters guaranteed: Filters remain visible on all tabs and queries stay explicit.
- Data fidelity plan defined (immutability, lineage, sampling/rollups): No changes to fidelity or rollups.
- Performance budgets defined for ingest and query latency at target scale: Budgets remain as specified in Technical Context.
- Security model defined (authn/authz, tenant isolation, data protection): Session-based auth with admin roles and mandatory password change.

## Phase 2: Planning Boundaries

Stop after contracts and documentation are produced. Task breakdown happens in `/speckit.tasks`.
