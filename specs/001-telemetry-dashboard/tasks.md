---

description: "Task list template for feature implementation"
---

# Tasks: OpenTelemetry Telemetry Dashboard

**Input**: Design documents from `/specs/001-telemetry-dashboard/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included to meet constitution requirements for contract and integration coverage.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Web app**: `backend/src/`, `frontend/src/`
- Paths shown below follow the project structure from plan.md

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create Go module scaffold in `backend/go.mod`
- [x] T002 Create SvelteKit scaffold in `frontend/package.json`
- [x] T003 [P] Add lint/test tooling configs in `backend/.golangci.yml`
- [x] T004 [P] Add local infra compose file in `deployments/docker-compose.yml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Define ClickHouse schema for telemetry in `backend/internal/storage/migrations/001_init.sql`
- [x] T006 Implement ClickHouse client in `backend/internal/storage/clickhouse.go`
- [x] T007 Implement config loader in `backend/internal/config/config.go`
- [x] T008 Implement auth/tenant middleware in `backend/internal/auth/middleware.go`
- [x] T009 Define telemetry domain models in `backend/internal/telemetry/models.go`
- [x] T010 Define query models in `backend/internal/query/models.go`
- [x] T011 Setup API router and middleware chain in `backend/internal/api/router.go`
- [x] T012 [P] Add shared API client wrapper in `frontend/src/services/api.ts`
- [x] T013 [P] Add app shell layout in `frontend/src/routes/+layout.svelte`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Explore telemetry with queries (Priority: P1) 🎯 MVP

**Goal**: Let users run explicit queries over logs, traces, and metrics with visual results

**Independent Test**: A user can run a query and see matching logs, traces, and metrics in the UI

### Tests for User Story 1 ⚠️

- [x] T014 [P] [US1] Contract test for `/api/query/run` in `backend/tests/contract/query_run_test.go`
- [x] T015 [P] [US1] Integration test for query flow in `frontend/tests/integration/query_flow.spec.ts`
- [x] T052 [P] [US1] Extend contract coverage for pagination metadata in `backend/tests/contract/query_run_test.go`
- [x] T053 [P] [US1] Extend integration test to cover quick time ranges and auto-refresh in `frontend/tests/integration/query_flow.spec.ts`

### Implementation for User Story 1

- [x] T016 [P] [US1] Implement log query builder in `backend/internal/query/builders/logs.go`
- [x] T017 [P] [US1] Implement trace query builder in `backend/internal/query/builders/traces.go`
- [x] T018 [P] [US1] Implement metric query builder in `backend/internal/query/builders/metrics.go`
- [x] T019 [US1] Implement query orchestration service in `backend/internal/query/service.go`
- [x] T020 [US1] Implement query handler in `backend/internal/api/handlers/query.go`
- [x] T021 [US1] Wire `/api/query/run` route in `backend/internal/api/router.go`
- [x] T022 [P] [US1] Implement query API client in `frontend/src/services/query.ts`
- [x] T023 [P] [US1] Add query state store in `frontend/src/lib/stores/query.ts`
- [x] T024 [US1] Build query form in `frontend/src/components/QueryForm.svelte`
- [x] T025 [US1] Build log results table in `frontend/src/components/LogResultsTable.svelte`
- [x] T026 [US1] Build trace results list in `frontend/src/components/TraceResultsList.svelte`
- [x] T027 [US1] Build metric chart component in `frontend/src/components/MetricChart.svelte`
- [x] T028 [US1] Assemble query page in `frontend/src/routes/query/+page.svelte`
- [x] T054 [US1] Add pagination params and metadata to query contract in `specs/001-telemetry-dashboard/contracts/query-api.yaml`
- [x] T055 [US1] Add page/limit fields to query models in `backend/internal/query/models.go`
- [x] T056 [US1] Apply limit/offset per signal query in `backend/internal/query/service.go`
- [x] T057 [US1] Surface pagination metadata in query response in `backend/internal/api/handlers/query.go`
- [x] T058 [P] [US1] Add pagination controls to log results in `frontend/src/components/LogResultsTable.svelte`
- [x] T059 [P] [US1] Add pagination controls to trace results in `frontend/src/components/TraceResultsList.svelte`
- [x] T060 [P] [US1] Add pagination controls to metrics view in `frontend/src/components/MetricChart.svelte`
- [x] T061 [US1] Add quick time range presets to query form in `frontend/src/components/QueryForm.svelte`
- [x] T062 [US1] Add custom date range selection to query form in `frontend/src/components/QueryForm.svelte`
- [x] T063 [US1] Add auto-refresh interval controls in `frontend/src/components/QueryForm.svelte`
- [x] T064 [US1] Implement auto-refresh scheduling in `frontend/src/lib/stores/query.ts`
- [x] T065 [US1] Wire pagination + refresh parameters in `frontend/src/services/query.ts`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Correlate signals across views (Priority: P2)

**Goal**: Pivot from traces to related logs and metrics without losing context

**Independent Test**: From a trace detail, a user can load related logs and metrics

### Tests for User Story 2 ⚠️

- [x] T029 [P] [US2] Contract test for `/api/traces/{traceId}/related` in `backend/tests/contract/trace_related_test.go`
- [x] T030 [P] [US2] Integration test for correlation pivot in `frontend/tests/integration/correlation_flow.spec.ts`

### Implementation for User Story 2

- [x] T031 [P] [US2] Implement correlation query builder in `backend/internal/query/correlation.go`
- [x] T032 [US2] Implement related telemetry service in `backend/internal/query/related_service.go`
- [x] T033 [US2] Implement trace related handler in `backend/internal/api/handlers/trace_related.go`
- [x] T034 [US2] Wire `/api/traces/{traceId}/related` route in `backend/internal/api/router.go`
- [x] T035 [P] [US2] Implement trace API client in `frontend/src/services/traces.ts`
- [x] T036 [US2] Add correlation panel component in `frontend/src/components/CorrelationPanel.svelte`
- [x] T037 [US2] Add trace detail page in `frontend/src/routes/traces/[traceId]/+page.svelte`
- [x] T038 [US2] Link trace results to detail view in `frontend/src/components/TraceResultsList.svelte`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Save and reuse queries (Priority: P3)

**Goal**: Save frequent queries and reuse or share them later

**Independent Test**: A user can save a query and rerun it from the saved list

### Tests for User Story 3 ⚠️

- [x] T039 [P] [US3] Contract tests for `/api/queries` in `backend/tests/contract/saved_queries_test.go`
- [x] T040 [P] [US3] Integration test for saved query reuse in `frontend/tests/integration/saved_queries_flow.spec.ts`

### Implementation for User Story 3

- [x] T041 [P] [US3] Add saved queries tables in `backend/internal/storage/migrations/002_saved_queries.sql`
- [x] T042 [US3] Implement saved query repository in `backend/internal/query/saved_queries.go`
- [x] T043 [US3] Implement saved query handlers in `backend/internal/api/handlers/saved_queries.go`
- [x] T044 [US3] Wire `/api/queries` routes in `backend/internal/api/router.go`
- [x] T045 [P] [US3] Implement saved queries API client in `frontend/src/services/saved_queries.ts`
- [x] T046 [US3] Build saved query list component in `frontend/src/components/SavedQueryList.svelte`
- [x] T047 [US3] Build saved queries page in `frontend/src/routes/queries/+page.svelte`
- [x] T048 [US3] Add save query action in `frontend/src/routes/query/+page.svelte`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T049 [P] Document query syntax and examples in `docs/query-syntax.md`
- [x] T050 [P] Update quickstart with env details in `specs/001-telemetry-dashboard/quickstart.md`
- [x] T051 Add release checklist in `docs/release-checklist.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 query results for pivot UI
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Independent of US2

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Foundational tasks marked [P] can run in parallel
- Contract tests for each story can run in parallel
- Query builders in US1 can run in parallel
- API client tasks in frontend can run in parallel with backend handlers

---

## Parallel Example: User Story 1

```bash
# Launch contract and integration tests in parallel:
Task: "Contract test for /api/query/run in backend/tests/contract/query_run_test.go"
Task: "Integration test for query flow in frontend/tests/integration/query_flow.spec.ts"

# Launch query builders in parallel:
Task: "Implement log query builder in backend/internal/query/builders/logs.go"
Task: "Implement trace query builder in backend/internal/query/builders/traces.go"
Task: "Implement metric query builder in backend/internal/query/builders/metrics.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
