# Quickstart: Secure Dashboard Access and UI Restyle

## Goal

Secure the dashboard with login, force the default admin to change password on first login, and manage users.

## Steps

1. Open the dashboard URL and log in with:
   - Username: admin
   - Password: admin
2. On first login, set a new password and continue to the dashboard.
3. Navigate to `/admin/users` to create additional users.
4. Verify new users can log in and see the Logs tab by default.

## Expected Result

- Unauthenticated users are redirected to the login screen.
- The default admin must change the password before seeing telemetry data.
- Users can navigate Logs, Metriche, and Tracce with the service dropdown set to "Tutti" by default and filters on the right.
