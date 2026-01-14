# Phase 0 Research: Secure Dashboard Access and UI Restyle

## Decision 1: Password storage for ClickHouse-backed users

- **Decision**: Store password hashes using bcrypt and a per-user salt; store hashes in a dedicated users table in ClickHouse.
- **Rationale**: bcrypt is widely supported in Go, resilient to brute-force attacks, and fits the current stack without adding a new database dependency.
- **Alternatives considered**: Argon2id (stronger but adds dependency and tuning complexity); introducing a new relational database (adds operational burden).

## Decision 2: Authentication mechanism

- **Decision**: Use a signed JWT stored in an HTTP-only, same-site cookie; validate token and user status on each request.
- **Rationale**: Simple integration with SvelteKit, avoids server-side session storage, and supports quick revocation by disabling user accounts.
- **Alternatives considered**: Server-side session table (more state and cleanup); basic auth (poor UX and security ergonomics).

## Decision 3: First-login password change enforcement

- **Decision**: Store a `must_change_password` flag on the user record and block all non-auth routes until it is cleared.
- **Rationale**: Explicit enforcement with a clear gate; easy to test and aligns with the requirement.
- **Alternatives considered**: Short-lived temporary password that expires (more complex edge cases and user frustration).

## Decision 4: UI layout pattern

- **Decision**: Use a left sidebar for navigation with a top-of-content service dropdown and a right filter rail; maintain the same filter controls per tab.
- **Rationale**: Clear navigation affordance and consistent filter placement; matches common dashboard expectations while allowing a modern restyle.
- **Alternatives considered**: Top nav tabs (less space for filters on dense query views); fully collapsible filter panel (hides critical controls).
