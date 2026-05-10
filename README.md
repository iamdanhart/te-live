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

To add a host user in production (requires `just flyproxy` running in another terminal):

```sh
just add-host-user-prod dan mysecretpasscode
```

---

## Database Migrations

Migrations are managed with [Liquibase](https://www.liquibase.com/) and live in `db/changelog/changes/`. The `Dockerfile.liquibase` image runs them.

To run migrations against production (requires `just flyproxy`):

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

## Deployment

The app runs on [Fly.io](https://fly.io) in the `ewr` region with a Fly Managed Postgres cluster.

```sh
just deploy        # fly deploy
just flyproxy      # open a local proxy to the prod DB on localhost:15432
```

`DATABASE_URL` and the managed Postgres credentials are stored as Fly secrets and are never committed.

---

## Security

| Layer | Mechanism |
|-------|-----------|
| Audience signups | IP-based rate limit (2-minute window) |
| Host POST routes | IP-based failure limit (blocks after 10 failed auth attempts in 15 minutes) |
| CSRF | Origin/Referer header checked against `allowed_hosts` |
| Host credentials | bcrypt (cost 12) |
| Framing | `X-Frame-Options: DENY` |
| MIME sniffing | `X-Content-Type-Options: nosniff` |
| Referrer | `Referrer-Policy: strict-origin` |

---

## TODOs

### Features
- **Host session invalidation** — Host auth uses HTTP Basic Auth with browser-cached credentials. There is no server-side session, so there is no way to forcibly sign out a host mid-session. Implementing this would require a sessions table and cookie-based auth.
- **Real favicon** — A placeholder emoji favicon is in use. A proper `.ico` file (mic icon, potentially commissioned) should be added to `router/static/` and the skip list in `router.go` updated.
- **OG image** — An `og:image` meta tag is stubbed out but commented in `base.html`. Needs an actual image asset.

### Deployment
- **Brewery custom domain** — Once the brewery picks a subdomain, add it to `config/prod.json` `allowed_hosts`, run `fly certs add <subdomain>`, and coordinate the DNS CNAME on their end.