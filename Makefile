# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=gommit
GIT=git

# Build parameters
BUILD_DIR=build
MAIN_FILE=cmd/gommit/main.go

VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build test clean deps lint run

all: clean deps build test

build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

test:
	$(GOTEST) -v ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

deps:
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	golangci-lint run

run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	./$(BUILD_DIR)/$(BINARY_NAME)

# Generate git commit message for current changes
commit-msg:
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: release
release:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	$(GIT) tag v$(VERSION)
	$(GIT) push origin v$(VERSION)