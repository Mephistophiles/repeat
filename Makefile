.PHONY: build test test-integration test-all vet lint clean install

BINARY := repeat
CMD_DIR := ./cmd/repeat
BUILD_DIR := ./build
VERSION ?= dev
LDFLAGS := -ldflags "-X main.buildVersion=$(VERSION)"

GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet

build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

test:
	$(GOTEST) ./internal/... -count=1 -race

test-integration:
	$(GOTEST) -tags=integration ./test/integration/ -v -count=1

test-all: test test-integration

vet:
	$(GOVET) ./...

lint:
	@golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed; run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

clean:
	rm -rf $(BUILD_DIR)
	$(GOCMD) clean -cache -testcache

install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) $(CMD_DIR)
