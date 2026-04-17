COMPOSE ?= docker compose

.PHONY: test test-unit test-integration build run docker-build up down logs demo-send demo health report

test:
	go test ./...

test-unit:
	go test ./...

test-integration:
	go test ./... -run Integration -count=1

build:
	go build ./...

run:
	go run ./cmd/server

docker-build:
	$(COMPOSE) build api

up:
	$(COMPOSE) up --build -d api

down:
	$(COMPOSE) down -v --remove-orphans

logs:
	$(COMPOSE) logs -f api

demo-send:
	$(COMPOSE) run --rm demo-sender

demo: up demo-send
	$(COMPOSE) logs api

health:
	@BASE_PATH="$${BASE_PATH:-/}"; \
	URL_PATH="$${BASE_PATH%/}/v1/manage/healthz"; \
	if [ -z "$${URL_PATH}" ]; then URL_PATH="/v1/manage/healthz"; fi; \
	curl -fsS "http://localhost:8080$${URL_PATH}"

report:
	@BASE_PATH="$${BASE_PATH:-/}"; \
	URL_PATH="$${BASE_PATH%/}/v1/reports"; \
	if [ -z "$${URL_PATH}" ]; then URL_PATH="/v1/reports"; fi; \
	curl -i -X POST \
	  -H 'Content-Type: application/reports+json' \
	  --data-binary @demo/reports.json \
	  "http://localhost:8080$${URL_PATH}"
