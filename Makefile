.PHONY: run build

run:
	go run .

build:
	go build -tags production -o te-live .