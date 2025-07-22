.PHONY: build test lint clean

APP=rdv
BIN=bin/$(APP)

# Build with version metadata
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      ?= $(shell git rev-parse --short HEAD)
DATE        ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     = -X github.com/yonasyiheyis/rdv/internal/version.Version=$(VERSION) \
              -X github.com/yonasyiheyis/rdv/internal/version.Commit=$(COMMIT) \
              -X github.com/yonasyiheyis/rdv/internal/version.Date=$(DATE)

build: ## Build for host OS/arch
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/rdv

test: ## Run unit tests
	go test ./...

test-ci: ## Run unit tests with coverage
	go test ./... -v -race -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint: ## Static analysis
	golangci-lint run

clean:
	rm -rf bin

docs:
	@bin/rdv help > DOCS.txt

ci: lint test-ci ## run linter and tests (used by GitHub Actions)