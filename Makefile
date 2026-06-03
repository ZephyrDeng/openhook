.PHONY: lint test build run tidy e2e ci clean

lint:
	test -z "$$(gofmt -l cmd internal)"
	go vet ./...

test:
	go test ./...

build:
	go build -o bin/openhook ./cmd/openhook

run:
	go run ./cmd/openhook

tidy:
	go mod tidy

e2e:
	scripts/local-e2e.sh

ci: lint test build e2e

clean:
	rm -rf bin dist coverage.out
