# Access Control and Tenant Isolation

## Authentication

- The dashboard requires authenticated access.
- Default admin credentials are provisioned and must be changed on first login.

## Authorization

- Admin users can create and manage additional users.
- Standard users can access logs, metrics, and traces views.

## Tenant Isolation

- Current deployments are single-tenant.
- Tenant context is fixed to a default tenant identifier for all requests.
