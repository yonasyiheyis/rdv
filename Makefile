.PHONY: build test lint clean

APP=rdv
BIN=bin/$(APP)
GOPATH ?= $(shell go env GOPATH)
GO1_26 ?= $(GOPATH)/bin/go1.26.0
GO ?= $(if $(wildcard $(GO1_26)),$(GO1_26),go)
GOTOOLCHAIN ?= go1.26.0
GOBIN ?= $(shell $(GO) env GOPATH)/bin
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint

# Build with version metadata
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      ?= $(shell git rev-parse --short HEAD)
DATE        ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     = -X github.com/yonasyiheyis/rdv/internal/version.Version=$(VERSION) \
              -X github.com/yonasyiheyis/rdv/internal/version.Commit=$(COMMIT) \
              -X github.com/yonasyiheyis/rdv/internal/version.Date=$(DATE)

build: ## Build for host OS/arch
	@mkdir -p bin
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/rdv

test: ## Run unit tests
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) test ./...

test-ci: ## Run unit tests with coverage
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) test ./... -v -race -coverprofile=coverage.out
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) tool cover -func=coverage.out

lint: ## Static analysis
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GOLANGCI_LINT) run

clean:
	rm -rf bin

docs:
	@bin/rdv help > DOCS.txt

ci: lint test-ci ## run linter and tests (used by GitHub Actions)

# Release helpers
release-patch:
	@NEW=$$(git tag --sort=-v:refname | head -1 | awk -F. '{printf "v%d.%d.%d",$$1+0,$$2+0,$$3+1}') && \
	echo "Tagging $$NEW" && \
	git tag -a $$NEW -m "Release $$NEW" && \
	git push origin $$NEW
