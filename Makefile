.PHONY: run run-nolimit run-noadmin run-noenforce build

run:
	-ENFORCE_SIGNUP_LIMIT=1 ENFORCE_ADMIN_AUTH=1 go run .

run-nolimit:
	-ENFORCE_ADMIN_AUTH=1 go run .

run-noadmin:
	-ENFORCE_SIGNUP_LIMIT=1 go run .

run-noenforce:
	-go run .

build:
	go build -tags production -o te-live .