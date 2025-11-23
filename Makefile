APP_NAME := pr-reviewer-service
CMD_DIR  := ./cmd/app

GO      := go
GOCMD   := $(GO)
GOFMT   := gofmt
GOTEST  := $(GO) test
GOBUILD := $(GO) build

DOCKER_IMAGE := pr-reviewer-service
DOCKER_COMPOSE := docker-compose

.PHONY: all generate fmt lint test build run docker-build up down logs

all: fmt lint test build

# --- codegen (oapi-codegen) ---

mocks:
	mockgen -source=internal/usecase/interfaces.go -destination=internal/mocks/mock_usecase.go -package=mocks
	mockgen -source=internal/repository/interfaces.go -destination=internal/mocks/mock_repo.go -package=mocks
	mockgen -source=db/transactor.go -destination=internal/mocks/mock_transactor.go -package=mocks

generate:
	oapi-codegen -package v1 -generate types  openapi/openapi.yml > internal/http/v1/dto_gen.go
	oapi-codegen -package v1 -generate server openapi/openapi.yml > internal/http/v1/server_gen.go

# --- formatting / lint / test ---

fmt:
	$(GOFMT) -w $$(find . -type f -name '*.go' -not -path './vendor/*')

lint:
	golangci-lint run ./...

test:
	$(GOTEST) ./...

# --- build / run ---

build:
	$(GOBUILD) -o bin/$(APP_NAME) $(CMD_DIR)

run:
	$(GO) run $(CMD_DIR)

# --- docker helpers ---

docker-build:
	docker build -t $(DOCKER_IMAGE) .

up:
	$(DOCKER_COMPOSE) up --build

down:
	$(DOCKER_COMPOSE) down

logs:
	$(DOCKER_COMPOSE) logs -f app