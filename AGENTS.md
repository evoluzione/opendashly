# opentelemetry-dashboard Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-13

## Active Technologies

- Go 1.22 (API), Node.js 20 LTS (SvelteKit) + OpenTelemetry Collector, ClickHouse, SvelteKit, uPlot, Go HTTP router (chi), OpenTelemetry semantic conventions (001-telemetry-dashboard)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

### Using Speckit

When a speckit command is invoked (e.g., `speckit specify`, `speckit plan`), the agent must:

1. Read the corresponding prompt file from `.codex/prompts/` (e.g., `.codex/prompts/speckit.specify.md` for `speckit specify`)
2. Follow the instructions and workflow defined in that prompt file
3. Execute the workflow as specified, reading from feature directories in `specs/` and `.specify/`

(Attention: pwsh path is 'c/Program Files/PowerShell/7/pwsh.exe')

#### Available Speckit Commands

- `speckit specify` - Create or update the feature specification from a natural language feature description. Read `.codex/prompts/speckit.specify.md` for workflow. Generates `spec.md` with user stories and acceptance criteria.

- `speckit clarify` - Identify and resolve underspecified areas in the feature spec. Read `.codex/prompts/speckit.clarify.md` for workflow. Encodes answers back into the spec before planning.

- `speckit plan` - Generate a technical implementation plan from the feature specification. Read `.codex/prompts/speckit.plan.md` for workflow. Creates `plan.md` with architecture, tech stack, and structure. Must run after `speckit specify`.

- `speckit tasks` - Break down the plan into actionable, dependency-ordered tasks. Read `.codex/prompts/speckit.tasks.md` for workflow. Generates `tasks.md` organized by user story and phase. Must run after `speckit plan`.

- `speckit implement` - Execute implementation tasks in phases according to `tasks.md`. Read `.codex/prompts/speckit.implement.md` for workflow. Manages task sequencing, prerequisites, and checklist validation.

- `speckit analyze` - Run cross-artifact consistency analysis on `spec.md`, `plan.md`, and `tasks.md`. Read `.codex/prompts/speckit.analyze.md` for workflow. Identifies inconsistencies, duplications, and ambiguities before implementation.

- `speckit checklist` - Generate domain-specific checklists to validate requirement quality and completeness. Read `.codex/prompts/speckit.checklist.md` for workflow. (Unit tests for requirements writing, not implementation testing).

- `speckit constitution` - Apply project constitution rules to ensure compliance with core principles. Read `.codex/prompts/speckit.constitution.md` for workflow. (OpenTelemetry-first ingestion, query-driven experience, data fidelity, performance, security).

## Code Style

Go 1.22 (API), Node.js 20 LTS (SvelteKit): Follow standard conventions

## Recent Changes

- 001-telemetry-dashboard: Added Go 1.22 (API), Node.js 20 LTS (SvelteKit) + OpenTelemetry Collector, ClickHouse, SvelteKit, uPlot, Go HTTP router (chi), OpenTelemetry semantic conventions

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
