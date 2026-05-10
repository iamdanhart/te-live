# te-live

Live band karaoke queue manager. Audience members sign up with their name and song choices; the host manages the queue from a separate authenticated view.

Built with Go, HTMX, and PostgreSQL. Deployed on Fly.io.

---

## Architecture

```
main.go               Entry point, HTTP server, graceful shutdown
config/               JSON-based config per environment (dev/production)
router/               HTTP handlers and route registration
  host_handlers.go    All /host/* routes
middleware/           Rate limiting, CSRF, security headers, admin auth
queue/                Queue interface and PostgreSQL implementation
grab_templates/       HTML templates (HTMX-driven)
db/changelog/         Liquibase migrations
cmd/add-host-user/    CLI tool to provision host credentials
```

The `Queue` interface (`queue/queue.go`) is the boundary between HTTP handlers and the database. The only implementation is `PgQueue`, backed by PostgreSQL.

---

## Local Development

### Prerequisites

- Go 1.26+
- Docker (for the local database)
- [`just`](https://github.com/casey/just) task runner
- A `.env` file for secrets (see [`.env.example`](.env.example))

### Start the database

```sh
just db-up       # start Postgres in Docker
just db-migrate  # run Liquibase migrations
```

### Run the server

```sh
just run       # uses config/dev.json — tweak that file to toggle enforcement
just run-prod  # uses config/prod.json from disk (auth + limits enforced); templates still served from disk
```

The server listens on `:8080`. The database defaults to `postgres://telive:telive@localhost:5432/telive`.

To override the database URL:

```sh
DATABASE_URL=postgres://... just run
```

### Reset the database

```sh
just db-reinit   # tears down volumes, restarts, and re-runs migrations
```

---

## Build System

The project uses Go build tags to switch between dev and production behaviour:

| Tag | Effect |
|-----|--------|
| _(none)_ | Templates and static files served from disk; config read from `config/dev.json` |
| `production` | Templates and static files embedded in the binary; config read from embedded `config/prod.json` |

```sh
just build   # runs tests then builds with -tags production
```

---

## Configuration

Config lives in `config/dev.json` and `config/prod.json`. Only `prod.json` is embedded in the binary at build time. Secrets (`DATABASE_URL`) are always read from environment variables.

| Field | Description |
|-------|-------------|
| `enforce_signup_limit` | Rate-limit audience signups by IP |
| `enforce_admin_auth` | Require passcode for host routes |
| `allowed_hosts` | Hosts allowed by the CSRF middleware (empty = allow all) |

---

## Host Authentication

The host view (`/host`) uses HTTP Basic Auth. Passcodes are stored as bcrypt hashes in `telive.host_users`.

To add a host user locally:

```sh
just add-host-user dan mysecretpasscode
```

To add a host user in production:

```sh
just add-host-user-prod dan mysecretpasscode
```

---

## Database Migrations

Migrations are managed with [Liquibase](https://www.liquibase.com/) and live in `db/changelog/changes/`. The `Dockerfile.liquibase` image runs them.

To run migrations against production:

```sh
just db-migrate-prod
```

All tables live under the `telive` schema.

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values. The file is gitignored and never committed.

| Variable | Used by | Where to find |
|----------|---------|---------------|
| `MPG_CLUSTER_ID` | `just flyproxy` | `fly mpg list` |
| `MPG_USER` | `just add-host-user-prod` | Fly dashboard > Postgres cluster > Connection details |
| `MPG_PASS` | `just add-host-user-prod` | Same as above; reset via `ALTER ROLE telive WITH PASSWORD '...'` |
| `MPG_MIGRATE_PASS` | `just db-migrate-prod` | Fly dashboard > Postgres cluster > Connection details (`liquibase-user`) |

---

## CI

Two workflows run on every push:

- **CI** (`.github/workflows/ci.yml`) — unit tests across `config`, `middleware`, and `router`
- **Integration Tests** (`.github/workflows/integration-test.yml`) — runs `./queue/...` using testcontainers (spins up Postgres and Liquibase in Docker)

Two workflows are triggered manually via the GitHub Actions UI:

- **Fly Deploy** (`.github/workflows/fly-deploy.yml`) — deploys the app to Fly.io
- **Liquibase Migrate (Prod)** (`.github/workflows/liquibase-migrate.yml`) — runs Liquibase migrations against the production database

### GitHub Secrets

| Secret | Used by | Notes |
|--------|---------|-------|
| `FLY_API_TOKEN` | Deploy, Liquibase Migrate | Generate with `fly tokens create deploy -x 999999h`; refresh by running that command and updating the secret |
| `MPG_MIGRATE_PASS` | Liquibase Migrate | Liquibase user password from the Fly Managed Postgres cluster |

---

## Query Generation (sqlc)

[sqlc](https://sqlc.dev) generates type-safe Go code from SQL queries. The generated files in `db/sqlcdb/` are committed — you only need the CLI if you're adding or changing queries.

```sh
brew install sqlc
```

Queries live in `db/queries/`. The schema used for generation is `db/schema.sql` (a plain SQL mirror of the Liquibase migrations — keep them in sync when adding tables). Config is in `sqlc.yaml`.

To regenerate after changing a query:

```sh
sqlc generate
```

**Current status:** `Songs()` has been migrated as a proof of concept. The remaining queries in `pg_queue.go` still use raw SQL and are candidates for future migration.

---

## Vendored Dependencies

HTMX is vendored at `router/static/vendor/htmx.min.js`. Two scripts in `dev_tools/` manage it:

```sh
dev_tools/check-htmx.sh    # outputs JSON comparing vendored vs latest stable version
dev_tools/vendor-htmx.sh   # downloads the latest stable release and replaces the vendored file
```

A manual GitHub Actions workflow (`.github/workflows/check-htmx.yml`) runs `check-htmx.sh` and fails if the vendored version is out of date.

---

## Deployment

The app runs on [Fly.io](https://fly.io) in the `ewr` region with a Fly Managed Postgres cluster.

```sh
just deploy        # fly deploy
```

`DATABASE_URL` and the managed Postgres credentials are stored as Fly secrets and are never committed.

---

## Security

| Layer | Mechanism |
|-------|-----------|
| Transport | HTTPS enforced by Fly (`force_https = true`) — Basic Auth depends on this; credentials would be exposed in plaintext on a plain HTTP deployment |
| Audience signups | IP-based rate limit (2-minute window) |
| Host POST routes | IP-based failure limit (blocks after 10 failed auth attempts in 15 minutes) |
| CSRF | Origin/Referer header checked against `allowed_hosts` |
| Host credentials | bcrypt (cost 12) |
| Input validation | Name required, max 50 chars; at least one song required; song IDs validated against the DB |
| Framing | `X-Frame-Options: DENY` |
| MIME sniffing | `X-Content-Type-Options: nosniff` |
| Referrer | `Referrer-Policy: strict-origin` |

---

## TODOs

### Features
- **Cookie-based host auth** — Host auth currently uses HTTP Basic Auth. Upgrading to a login form with signed `HttpOnly; Secure; SameSite=Strict` cookies would add: automatic session expiry (e.g. 8 hours), a working logout endpoint, and no credentials sent on every request. Requires a `sessions` table, a login page, and a few new routes.
- **Real favicon** — A placeholder emoji favicon is in use. A proper `.ico` file (mic icon, potentially commissioned) should be added to `router/static/` and the skip list in `router.go` updated.
- **OG image** — An `og:image` meta tag is stubbed out but commented in `base.html`. Needs an actual image asset.

### Data
- **Tab URLs** — `tab_url` is only populated for Bohemian Rhapsody (PoC). Remaining songs need their URLs updated directly in the DB.

### sqlc Migration
- **Migrate remaining queries** — Only `Songs()` has been moved to sqlc. The remaining queries in `pg_queue.go` are candidates, though some (e.g. dynamic position subqueries, multi-step transactions) will need care. `db/schema.sql` must also be kept in sync with any new Liquibase migrations.

### Known Limitations
- **Float position drift** — `MoveEntry` uses a midpoint algorithm (`(a + b) / 2`) to reorder queue entries without renumbering. After many drag-drops in one session, positions can converge toward float64 precision limits, causing two entries to collide on the same value. Not a practical problem at current queue lengths, but a periodic rebalance (e.g. reassign integer positions 1, 2, 3… on `ToggleSignups` open) would eliminate the risk.
- **`times_on_stage` is tracked but not displayed** — `CompleteCurrentSong` increments the counter but nothing reads it in templates or handlers. Could feed a fairness indicator in the host view ("sang 3 times tonight").
- **Shallow health check** — `GET /health` returns 200 immediately with no DB ping. A DB blip won't cause a machine restart (intentional), but UptimeRobot will show green while all DB-backed requests are failing.

### Nice to Have
- **Song catalog management** — Songs are currently managed via direct SQL. A host-only UI for adding and removing songs would make setlist changes self-serve without needing DB access.
- **Uptime monitoring** — Fly has no built-in alerting. Point a free UptimeRobot monitor at `/health` to get email notifications if the app goes down.

### Deployment
- **Brewery custom domain** — Once the brewery picks a subdomain, add it to `config/prod.json` `allowed_hosts`, run `fly certs add <subdomain>`, and coordinate the DNS CNAME on their end.