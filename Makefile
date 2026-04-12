.PHONY: run run-nolimit build

run:
	ENFORCE_SIGNUP_LIMIT=1 go run .

run-nolimit:
	go run .

build:
	go build -tags production -o te-live .