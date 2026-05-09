db_url := env("DB_URL", "postgres://telive:telive@localhost:5432/telive")
db_schema := env("DB_SCHEMA", "telive")

run:
    -DATABASE_URL={{db_url}} DB_SCHEMA={{db_schema}} ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

run-nolimit:
    -DATABASE_URL={{db_url}} DB_SCHEMA={{db_schema}} ENFORCE_ADMIN_AUTH=1 go run .

run-noadmin:
    -DATABASE_URL={{db_url}} DB_SCHEMA={{db_schema}} ENFORCE_SIGNUP_LIMIT=1 go run .

run-noenforce:
    -DATABASE_URL={{db_url}} DB_SCHEMA={{db_schema}} go run .

run-prod:
    -DATABASE_URL={{db_url}} DB_SCHEMA={{db_schema}} ENV=production ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

build:
    CGO_ENABLED=0 go test ./...
    CGO_ENABLED=0 go build -tags production -o te-live .

db-up:
    docker compose up -d db

db-migrate:
    docker compose run --rm --build liquibase

db-migrate-prod:
    source .env && docker compose run --rm --build liquibase \
      --url=$MPG_URL --username=$MPG_USER --password=$MPG_PASS \
      --defaultSchemaName=$MPG_SCHEMA --liquibaseSchemaName=$MPG_LB_SCHEMA \
      --changeLogFile=root.yaml update

db-down:
    docker compose down

db-down-clear-vol:
    docker compose down -v

db-reinit: db-down-clear-vol db-up db-migrate