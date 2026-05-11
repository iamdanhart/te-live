DATABASE_URL := env("DATABASE_URL", "postgres://telive:telive@localhost:5432/telive")

run:
    -DATABASE_URL={{DATABASE_URL}} go run .

# Runs with prod config (auth + rate limiting enforced, allowed_hosts active) but
# templates and static files are still served from disk, not the embedded binary.
run-prod:
    -DATABASE_URL={{DATABASE_URL}} ENV=production go run .

test:
    CGO_ENABLED=0 go vet ./...
    CGO_ENABLED=0 go test ./config/... ./middleware/... ./router/...

# CGO_ENABLED is intentionally omitted — -race requires CGO
itest:
    go test -race ./queue/...

build:
    CGO_ENABLED=0 go test ./...
    CGO_ENABLED=0 go build -tags production -o te-live .

db-up:
    docker compose up -d db

db-migrate:
    docker compose run --rm --build liquibase

add-host-user label passcode:
    @DATABASE_URL={{DATABASE_URL}} go run ./cmd/add-host-user -label={{label}} -passcode={{passcode}}

add-host-user-prod label passcode:
    @source .env && fly mpg proxy $MPG_CLUSTER_ID -p 15432 & \
    PROXY_PID=$! && \
    sleep 2 && \
    DATABASE_URL="postgresql://$MPG_USER:$MPG_PASS@localhost:15432/telive?sslmode=disable" go run ./cmd/add-host-user -label={{label}} -passcode={{passcode}}; \
    kill $PROXY_PID

test-add-users:
    hurl dev_tools/add_users_to_queue.hurl

test-toggle-signups:
    hurl dev_tools/toggle_signups.hurl

generate-qr:
    source .env && qrencode -t SVG -o docs/qr.svg "$APP_URL"

deploy:
    fly deploy

# For a full prod reinit: drop and recreate the telive database in the Fly dashboard first, then run this.
# Note: host.docker.internal resolves on macOS/Windows Docker Desktop only.
# On Linux, pass --add-host=host.docker.internal:host-gateway to docker run instead.
db-migrate-prod:
    source .env && fly mpg proxy $MPG_CLUSTER_ID -p 15432 & \
    PROXY_PID=$! && \
    sleep 2 && \
    docker build -f Dockerfile.liquibase -t te-live-liquibase . && docker run --rm te-live-liquibase \
      --url="jdbc:postgresql://host.docker.internal:15432/telive?sslmode=disable" \
      --username=liquibase-user --password=$MPG_MIGRATE_PASS \
      --defaultSchemaName=telive --liquibaseSchemaName=public \
      --search-path=/liquibase/changelog --changeLogFile=root.yaml update; \
    kill $PROXY_PID

db-down:
    docker compose down

db-down-clear-vol:
    docker compose down -v

db-reinit: db-down-clear-vol db-up db-migrate