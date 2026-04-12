.PHONY: run run-nolimit run-noadmin run-noenforce run-prod build db-up db-migrate db-down

run:
	-ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

run-nolimit:
	-ENFORCE_ADMIN_AUTH=1 go run .

run-noadmin:
	-ENFORCE_SIGNUP_LIMIT=1 go run .

run-noenforce:
	-go run .

run-prod:
	-ENV=production ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

build:
	go build -tags production -o te-live .

db-up:
	docker compose up -d db

db-migrate:
	docker compose run --rm --build liquibase

db-down:
	docker compose down