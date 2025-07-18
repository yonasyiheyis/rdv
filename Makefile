.PHONY: build test lint clean

APP=rdv
BIN=bin/$(APP)

build: ## Build for host OS/arch
	@mkdir -p bin
	go build -o $(BIN) ./cmd/...

test: ## Run unit tests
	go test ./...

lint: ## Static analysis
	golangci-lint run

clean:
	rm -rf bin
