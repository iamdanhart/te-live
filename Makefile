.PHONY: run run-nolimit run-noadmin run-noenforce run-prod build db-up db-migrate db-down db-down-clear-vol db-reinit

DB_URL ?= postgres://telive:telive@localhost:5432/telive

run:
	-DATABASE_URL=$(DB_URL) ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

run-nolimit:
	-DATABASE_URL=$(DB_URL) ENFORCE_ADMIN_AUTH=1 go run .

run-noadmin:
	-DATABASE_URL=$(DB_URL) ENFORCE_SIGNUP_LIMIT=1 go run .

run-noenforce:
	-DATABASE_URL=$(DB_URL) go run .

run-prod:
	-DATABASE_URL=$(DB_URL) ENV=production ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

build:
	go build -tags production -o te-live .

db-up:
	docker compose up -d db

db-migrate:
	docker compose run --rm --build liquibase

db-down:
	docker compose down

db-down-clear-vol:
	docker compose down -v

db-reinit: db-down-clear-vol db-up db-migrate