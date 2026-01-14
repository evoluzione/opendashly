# Data Model: Secure Dashboard Access and UI Restyle

## Entities

### User

- **Fields**:
  - `id` (unique identifier)
  - `username` (unique, required)
  - `password_hash` (bcrypt hash)
  - `role` (enum: `admin`, `user`)
  - `must_change_password` (boolean)
  - `is_disabled` (boolean)
  - `created_at` (timestamp)
  - `last_login_at` (timestamp, nullable)
- **Validation rules**:
  - `username` is required and unique
  - `password_hash` is required
  - `role` must be one of `admin`, `user`
- **State transitions**:
  - `must_change_password` true -> false after successful password change
  - `is_disabled` false -> true when admin disables a user

### AuthToken (logical)

- **Fields**:
  - `user_id`
  - `role`
  - `issued_at`
  - `expires_at`
- **Notes**: Token is stored client-side in a secure cookie; not persisted in storage.

### Service (derived)

- **Fields**:
  - `name` (string)
- **Notes**: Derived from telemetry resource attribute `service.name` and used for the service dropdown.

## Relationships

- A **User** can authenticate and receive an **AuthToken**.
- **Admin** users can create and manage other **Users**.
- **Services** are independent; used only for filtering queries.
