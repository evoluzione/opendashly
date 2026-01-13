# Implementation Plan: OpenTelemetry Telemetry Dashboard

**Branch**: `001-telemetry-dashboard` | **Date**: 2026-01-13 | **Spec**: /mnt/c/Progetti/opentelemetry-dashboard/specs/001-telemetry-dashboard/spec.md
**Input**: Feature specification from `/specs/001-telemetry-dashboard/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a unified observability dashboard that ingests OpenTelemetry logs, traces, and metrics via an OpenTelemetry Collector, stores telemetry in ClickHouse, exposes query and correlation APIs in Go, and renders interactive visualizations in a SvelteKit UI using uPlot.

## Technical Context

**Language/Version**: Go 1.22 (API), Node.js 20 LTS (SvelteKit)  
**Primary Dependencies**: OpenTelemetry Collector, ClickHouse, SvelteKit, uPlot, Go HTTP router (chi), OpenTelemetry semantic conventions  
**Storage**: ClickHouse 24.x  
**Testing**: Go test, Testcontainers for ClickHouse, Vitest for UI, Playwright for end-to-end flows  
**Target Platform**: Linux servers for backend/collector, modern browsers for frontend  
**Project Type**: web app  
**Performance Goals**: ingest 50k spans/sec, 20k log lines/sec, 5k metric series/sec; 95% of user queries within 2s  
**Constraints**: p95 API latency under 1s for typical queries; UI renders charts in under 500ms after data load  
**Scale/Scope**: 100 services, 30-day retention, 200 concurrent users

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- OTLP ingest path specified; schema preservation and any normalization documented
- Query model defined; saved queries and visibility of filters guaranteed
- Data fidelity plan defined (immutability, lineage, sampling/rollups)
- Performance budgets defined for ingest and query latency at target scale
- Security model defined (authn/authz, tenant isolation, data protection)

**Gate Status**: PASS. All principles addressed in Technical Context and Phase 1 design outputs.

## Project Structure

### Documentation (this feature)

```text
specs/001-telemetry-dashboard/
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
│   ├── ingest/
│   ├── query/
│   ├── storage/
│   └── auth/
└── tests/
    ├── contract/
    ├── integration/
    └── unit/

frontend/
├── src/
│   ├── routes/
│   ├── lib/
│   ├── components/
│   └── services/
└── tests/
    ├── integration/
    └── unit/
```

**Structure Decision**: Web application with separate `backend/` and `frontend/` directories to isolate Go APIs from SvelteKit UI.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
