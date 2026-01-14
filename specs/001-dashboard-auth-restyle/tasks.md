# Tasks: Secure Dashboard Access and UI Restyle

**Input**: Design documents from `/specs/001-dashboard-auth-restyle/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Required by constitution for contract coverage and query-to-visualization integration flows; test tasks included below.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Add auth configuration fields and defaults in `backend/internal/config/config.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Add users/auth ClickHouse migration in `backend/internal/storage/migrations/003_users.sql`
- [X] T003 Register auth migration in `backend/internal/storage/migrations.go`
- [X] T004 [P] Create user model definitions in `backend/internal/auth/models.go`
- [X] T005 [P] Implement password hashing helpers in `backend/internal/auth/password.go`
- [X] T006 [P] Implement JWT helpers in `backend/internal/auth/jwt.go`
- [X] T007 Implement user repository in `backend/internal/auth/repo.go`
- [X] T008 Update auth middleware for JWT cookies, role checks, and must-change-password gate in `backend/internal/auth/middleware.go`
- [X] T009 Seed default admin on startup in `backend/cmd/api/main.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Secure access to the dashboard (Priority: P1) 🎯 MVP

**Goal**: Require login for dashboard access and enforce first-login password change for default admin.

**Independent Test**: Open dashboard unauthenticated to confirm redirect, then log in as admin, change password, and access the dashboard.

### Implementation for User Story 1

- [X] T010 [P] [US1] Add contract tests for auth endpoints in `backend/tests/contract/auth_test.go`
- [X] T011 [P] [US1] Add integration test for login + first-login password change flow in `backend/tests/integration/auth_flow_test.go`
- [X] T012 [US1] Add auth handlers (login/logout/change-password) in `backend/internal/api/handlers/auth.go`
- [X] T013 [US1] Wire auth routes and unauth allowlist in `backend/internal/api/router.go`
- [X] T014 [P] [US1] Add auth session store in `frontend/src/lib/stores/auth.ts`
- [X] T015 [P] [US1] Add auth API client in `frontend/src/services/auth.ts`
- [X] T016 [US1] Update API fetch to include credentials in `frontend/src/services/api.ts`
- [X] T017 [US1] Create login page in `frontend/src/routes/login/+page.svelte`
- [X] T018 [US1] Create first-login password change page in `frontend/src/routes/first-login/+page.svelte`
- [X] T019 [US1] Add auth guard and redirects in `frontend/src/routes/+layout.svelte`

**Checkpoint**: User Story 1 should now be fully functional and testable independently

---

## Phase 4: User Story 2 - Admin creates additional users (Priority: P2)

**Goal**: Provide admin-only user management to create additional users.

**Independent Test**: Log in as admin, create a user, then log in as the new user and access the dashboard.

### Implementation for User Story 2

- [X] T020 [P] [US2] Add contract tests for user management endpoints in `backend/tests/contract/users_test.go`
- [X] T021 [P] [US2] Add integration test for admin create-user flow in `backend/tests/integration/user_management_flow_test.go`
- [X] T022 [US2] Add admin user handlers (list/create) in `backend/internal/api/handlers/users.go`
- [X] T023 [US2] Wire user management routes in `backend/internal/api/router.go`
- [X] T024 [P] [US2] Add users API client in `frontend/src/services/users.ts`
- [X] T025 [P] [US2] Create admin user management page in `frontend/src/routes/admin/users/+page.svelte`
- [X] T026 [US2] Build user management form/table component in `frontend/src/components/UserManagement.svelte`

**Checkpoint**: User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Navigate a modernized dashboard layout (Priority: P3)

**Goal**: Restyle the dashboard with a sidebar, default Logs tab, service dropdown, and right-side filters.

**Independent Test**: Log in, verify Logs is default, switch tabs, select services, and confirm filters stay visible on the right.

### Implementation for User Story 3

- [X] T027 [P] [US3] Add contract test for services endpoint in `backend/tests/contract/services_test.go`
- [X] T028 [P] [US3] Add integration test for service filter + query run flow in `backend/tests/integration/service_filter_flow_test.go`
- [X] T029 [US3] Add service listing query in `backend/internal/query/services.go`
- [X] T030 [US3] Add services handler in `backend/internal/api/handlers/services.go`
- [X] T031 [US3] Wire services route in `backend/internal/api/router.go`
- [X] T032 [P] [US3] Create sidebar navigation component in `frontend/src/components/Sidebar.svelte`
- [X] T033 [P] [US3] Create service dropdown component and store wiring in `frontend/src/components/ServiceDropdown.svelte` and `frontend/src/lib/stores/query.ts`
- [X] T034 [US3] Restyle dashboard layout with sidebar + tabs in `frontend/src/routes/+page.svelte`
- [X] T035 [US3] Align filter rail layout in `frontend/src/components/QueryForm.svelte`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T036 [P] Add empty-state UX for no services in `frontend/src/components/ServiceDropdown.svelte`
- [X] T037 Document latency budgets in `docs/performance-budgets.md`
- [X] T038 Document data fidelity rules in `docs/data-fidelity.md`
- [X] T039 Document access control and tenant isolation in `docs/access-control.md`
- [X] T040 Update query visibility rules in `docs/query-syntax.md`
- [X] T041 Document data retention expectations in `docs/data-retention.md`
- [X] T042 Validate quickstart steps and update notes in `specs/001-dashboard-auth-restyle/quickstart.md`

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
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Integrates with US1 auth but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Independent of US2; uses auth from US1

### Within Each User Story

- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- Foundational tasks T004, T005, T006 can run in parallel
- User Story 1 tasks T014 and T015 can run in parallel
- User Story 2 tasks T024 and T025 can run in parallel
- User Story 3 tasks T032 and T033 can run in parallel

---

## Parallel Example: User Story 1

```bash
Task: "Add auth session store in frontend/src/lib/stores/auth.ts"
Task: "Add auth API client in frontend/src/services/auth.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. STOP and validate User Story 1 independently

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
