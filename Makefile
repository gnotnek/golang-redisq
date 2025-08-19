BIN_DIR := ./bin
APP := $(BIN_DIR)/redisq

# Default build target
.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	go build -o $(APP) ./cmd

# Run worker (uses cobra CLI)
.PHONY: run-worker
run-worker: build
	$(APP) worker

# Run API
.PHONY: run-api
run-api: build
	$(APP) api

# Run tests
.PHONY: test
test:
	go test ./... -cover

# Run linter (needs golangci-lint installed)
.PHONY: lint
lint:
	golangci-lint run ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
