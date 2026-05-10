# Plan: Production Deployment

## Steps

1. Commit all pending changes locally
2. `fly launch` — detect the Go app, generate `fly.toml` and `Dockerfile`
3. Set Fly secrets:
   - `fly secrets set DATABASE_URL=<postgres connection string from Fly dashboard>`
   - `fly secrets set DB_SCHEMA=telive`
   - `fly secrets set ENFORCE_ADMIN_AUTH=1`
   - `fly secrets set ENV=production`
4. `fly deploy` — build and deploy the app
5. In a separate terminal, start the Fly proxy: `just flyproxy`
6. `just db-migrate-prod` — apply all migrations to the Fly Managed Postgres database
7. `just add-host-user-prod dan <passcode>` — create your host user
8. Verify `/health` returns 200
9. Verify `/host` prompts for credentials and accepts your passcode
10. Verify `/` loads the queue page
11. Verify a test signup works end-to-end

## Notes

- `just flyproxy` must be running before any prod DB operations (`db-migrate-prod`, `add-host-user-prod`)
- The proxy will fail to start if port 5432 is already in use — stop the local DB first with `just db-down`
- For a full prod reinit: drop and recreate the `telive` database in the Fly dashboard, then repeat steps 5-7
- `DATABASE_URL` in step 3 should be the pooler URL from the Fly dashboard, with `?search_path=telive` appended