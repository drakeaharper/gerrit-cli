# Makefile for Gerrit CLI (gerry)

# Binary name
BINARY_NAME=gerry
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOINSTALL=$(GOCMD) install

# Build variables
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Output directory
OUTPUT_DIR=bin

all: build

build:
	mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) -v ./cmd/gerry

clean:
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -v -cover -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

deps:
	$(GOMOD) download
	$(GOMOD) tidy

install: build
	$(GOINSTALL) ./cmd/gerry

# Cross compilation
build-linux:
	mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 -v ./cmd/gerry

build-windows:
	mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe -v ./cmd/gerry

build-darwin:
	mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 -v ./cmd/gerry
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 -v ./cmd/gerry

build-all: build-linux build-windows build-darwin

# Development helpers
run:
	$(GOCMD) run ./cmd/gerry

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

lint:
	golangci-lint run

.PHONY: all build clean test test-coverage deps install build-linux build-windows build-darwin build-all run fmt vet lint