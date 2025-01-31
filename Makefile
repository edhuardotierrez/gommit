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

# Version from VERSION file
VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Supported platforms (matching release.yml and install.sh)
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: all build test clean deps lint run release-builds

all: clean deps build test

# Build for current platform
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

# Build for all supported platforms
release-builds:
	$(foreach PLATFORM,$(PLATFORMS), \
		$(eval OS := $(word 1,$(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(PLATFORM)))) \
		mkdir -p $(BUILD_DIR)/$(OS)_$(ARCH) && \
		GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) \
			-o $(BUILD_DIR)/$(BINARY_NAME)_$(OS)_$(ARCH) $(MAIN_FILE) && \
	)

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

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Generate git commit message for current changes
commit-msg: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Create and push a new release tag
release: clean test release-builds
	$(GIT) tag v$(VERSION)
	$(GIT) push origin v$(VERSION)

# Install locally
install: build
	mv $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

.PHONY: release install