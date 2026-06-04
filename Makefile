.PHONY: lint test frontend-build build build-all run run-dev bootstrap deploy deploy-production production-smoke production-readiness tidy e2e deploy-openhook-test deploy-production-test provider-smoke-test qq-token-smoke-test qq-c2c-smoke-test production-smoke-test production-readiness-test frontend-nav-test mobile-layout-test frontend-content-test bootstrap-test ci clean

lint:
	test -z "$$(gofmt -l cmd internal)"
	go vet ./...

test:
	go test ./...

frontend-build:
	cd frontend && npm run build

build: frontend-build
	go build -o bin/openhook ./cmd/openhook

build-all: build

run: build
	./bin/openhook

run-dev:
	go run ./cmd/openhook

deploy:
	scripts/deploy-openhook.sh

deploy-production:
	scripts/deploy-production.sh

production-smoke:
	scripts/production-smoke.sh

production-readiness:
	scripts/production-readiness.sh

bootstrap:
	scripts/bootstrap-openhook-server.sh

tidy:
	go mod tidy

e2e:
	scripts/local-e2e.sh

deploy-openhook-test:
	scripts/deploy-openhook-test.sh

deploy-production-test:
	scripts/deploy-production-test.sh

provider-smoke-test:
	scripts/provider-smoke-test.sh

qq-token-smoke-test:
	scripts/qq-token-smoke-test.sh

qq-c2c-smoke-test:
	scripts/qq-c2c-smoke-test.sh

production-smoke-test:
	scripts/production-smoke-test.sh

production-readiness-test:
	scripts/production-readiness-test.sh

frontend-nav-test:
	scripts/frontend-nav-test.sh

mobile-layout-test:
	scripts/mobile-layout-test.sh

frontend-content-test:
	scripts/frontend-content-test.sh

bootstrap-test:
	scripts/bootstrap-openhook-server-test.sh

ci: lint test frontend-nav-test mobile-layout-test frontend-content-test build e2e

clean:
	rm -rf bin dist coverage.out
	cd frontend && rm -rf node_modules dist
