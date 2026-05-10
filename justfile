db_url := env("DB_URL", "postgres://telive:telive@localhost:5432/telive")

run:
    -DATABASE_URL={{db_url}} ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

run-nolimit:
    -DATABASE_URL={{db_url}} ENFORCE_ADMIN_AUTH=1 go run .

run-noadmin:
    -DATABASE_URL={{db_url}} ENFORCE_SIGNUP_LIMIT=1 go run .

run-noenforce:
    -DATABASE_URL={{db_url}} go run .

run-prod:
    -DATABASE_URL={{db_url}} ENV=production ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

build:
    CGO_ENABLED=0 go test ./...
    CGO_ENABLED=0 go build -tags production -o te-live .

db-up:
    docker compose up -d db

db-migrate:
    docker compose run --rm --build liquibase

add-host-user label passcode:
    @DATABASE_URL={{db_url}} go run ./cmd/add-host-user -label={{label}} -passcode={{passcode}}

add-host-user-prod label passcode:
    @source .env && DATABASE_URL="postgresql://$MPG_USER:$MPG_PASS@localhost:15432/telive?sslmode=disable" go run ./cmd/add-host-user -label={{label}} -passcode={{passcode}}

test-add-users:
    hurl dev_tools/add_users_to_queue.hurl

flyproxy:
    # Creates a local proxy to the Fly Managed Postgres cluster on localhost:15432.
    # Run this in a separate terminal before db-migrate-prod or add-host-user-prod.
    fly mpg proxy 82ylg01lgmmrzx19 -p 15432

# For a full prod reinit: drop and recreate the telive database in the Fly dashboard first, then run this.
# Requires flyproxy running in another terminal.
db-migrate-prod:
    source .env && docker build -f Dockerfile.liquibase -t te-live-liquibase . && docker run --rm te-live-liquibase \
      --url="jdbc:postgresql://host.docker.internal:15432/telive?sslmode=disable" \
      --username=$MPG_MIGRATE_USER --password=$MPG_MIGRATE_PASS \
      --defaultSchemaName=$MPG_SCHEMA --liquibaseSchemaName=$MPG_LB_SCHEMA \
      --search-path=/liquibase/changelog --changeLogFile=root.yaml update

db-down:
    docker compose down

db-down-clear-vol:
    docker compose down -v

db-reinit: db-down-clear-vol db-up db-migrate