# AGENTS.md

Guidelines for agents working in this repository.

## Purpose
- Provide consistent behavior for human or automated agents.
- Reduce errors, regressions, and release surprises.
- Keep changes reproducible and verifiable.

## General principles
- Prefer small, focused, reviewable changes.
- Always explain why a change is needed, not just what changed.
- Keep context minimal: touch only necessary files.
- Avoid new dependencies unless strictly required.

## Workflow
- Understand the problem: read relevant files before coding.
- Plan: define clear steps when work is non-trivial.
- Implement: apply incremental, coherent changes.
- Verify: run targeted tests or checks when possible.
- Document: update README or notes only if behavior changes.

## Code changes
- Follow the existing project style.
- Avoid unrequested changes (global formatting, unnecessary refactors).
- Preserve compatibility across components (backend, frontend, collector).
- Add comments only when they clarify non-obvious logic.

## Testing and validation
- Prefer tests targeted to the modified areas.
- If you cannot test, explain why and what you would verify.
- Avoid flaky tests or those depending on external networks.

## Git and releases
- Do not rewrite repository history.
- Do not delete or revert others' changes unless explicitly asked.
- Keep commits logical with descriptive messages.

## Security and data
- Do not include secrets or credentials in code or logs.
- Consider impact on sensitive data and external service access.

## Communication
- If requirements are missing, ask short, focused questions.
- Report risks, assumptions, and limitations clearly.
- Suggest next steps (tests, build, deploy) when relevant.

## Repository layout
- backend: services and application logic
- frontend: user interface
- docker/otel-collector/collector-config.yaml and docker-compose.yml: local configuration

