SHELL := /bin/bash
.DEFAULT_GOAL := help

# build settings; release builds pass GOOS=linux and each GOARCH
GOOS    ?= $(shell go env GOOS)
GOARCH  ?= $(shell go env GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST    := dist
BINARY  := browser-reporting-api
LDFLAGS := -s -w -X main.version=$(VERSION)

# tool versions, run with "go run" so they are built with this module's go
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK   := go run golang.org/x/vuln/cmd/govulncheck@v1.9.0

# the race detector on macos needs this linker flag with some xcode versions
TEST_ENV   := $(if $(filter Darwin,$(shell uname -s)),CGO_LDFLAGS=-w,)
TEST_FLAGS := -race -count=1

# local runs
COMPOSE  ?= docker compose
BASE_URL ?= http://localhost:8080
URL_PATH  = $(patsubst %/,%,$(or $(BASE_PATH),/))

.PHONY: help
help: ## show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ build

.PHONY: build
build: ## build the server binary (dist/browser-reporting-api) for GOOS and GOARCH
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) ./cmd/server

.PHONY: image
image: ## build the container image (browser-reporting-api:dev)
	docker build --build-arg VERSION=$(VERSION) -t $(BINARY):dev .

.PHONY: clean
clean: ## remove build output
	rm -rf $(DIST) coverage.out

##@ test

.PHONY: test
test: ## run all tests with the race detector
	$(TEST_ENV) go test $(TEST_FLAGS) ./...

.PHONY: test-unit
test-unit: ## run unit tests
	$(TEST_ENV) go test $(TEST_FLAGS) ./internal/...

.PHONY: cover
cover: ## run tests with coverage and print a summary
	go test -count=1 -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -func=coverage.out | tail -1

##@ quality

.PHONY: lint
lint: ## run golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: fmt
fmt: ## format code
	$(GOLANGCI_LINT) fmt ./...

.PHONY: vuln
vuln: ## check for known vulnerabilities
	$(GOVULNCHECK) ./...

.PHONY: tidy
tidy: ## tidy go.mod and go.sum
	go mod tidy

.PHONY: check
check: lint test vuln ## run lint, tests, and the vulnerability check, as CI does

##@ local runs

.PHONY: run
run: ## run the server locally with go run
	go run ./cmd/server

.PHONY: up
up: ## start the server in docker compose and wait until it is healthy
	$(COMPOSE) up --build --detach --wait api

.PHONY: down
down: ## stop docker compose
	$(COMPOSE) down --volumes --remove-orphans

.PHONY: logs
logs: ## follow the server's output
	$(COMPOSE) logs --follow api

.PHONY: demo
demo: up ## start the server, send the demo reports, and print the output
	$(COMPOSE) run --rm demo-sender
	$(COMPOSE) logs api

.PHONY: report
report: ## send the demo reports to a running server (BASE_URL, BASE_PATH)
	curl -fsS -X POST -H 'Content-Type: application/reports+json' \
	  --data-binary @demo/reports.json "$(BASE_URL)$(URL_PATH)/v1/reports"

.PHONY: health
health: ## check a running server's health endpoint (BASE_URL, BASE_PATH)
	curl -fsS "$(BASE_URL)$(URL_PATH)/v1/manage/healthz"
