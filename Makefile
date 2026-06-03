.PHONY: test build run tidy

test:
	go test ./...

build:
	go build -o bin/openhook ./cmd/openhook

run:
	go run ./cmd/openhook

tidy:
	go mod tidy
