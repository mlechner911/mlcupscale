# Image Upscale Service
# Copyright (c) 2026 Michael Lechner

# Variables
APP_NAME := upscale-service
VERSION := 1.0.0
BUILD_DIR := build
SERVER_BIN := $(BUILD_DIR)/upscale-server
CLIENT_BIN := $(BUILD_DIR)/upscale-client

# Docker variables
DOCKER_IMAGE := upscale-service
DOCKER_TAG := latest
DOCKER_COMPOSE := docker compose -f deployments/docker/docker-compose.yml

# Go variables
GO_FILES := $(shell find . -name '*.go')

.PHONY: all build build-server build-client clean run run-docker stop-docker test test-integration models help git-init

# Default target
all: help

## build: Build both server and client binaries
build: build-server build-client

## build-server: Build the server binary
build-server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(SERVER_BIN) ./cmd/server

## build-client: Build the client binary
build-client:
	@echo "Building client..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(CLIENT_BIN) ./cmd/client

## clean: Remove build artifacts and temporary files
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f test_upscaled.png test_output.png
	rm -rf test/testdata/sample_images/test.png

## run: Run the server locally (requires models and ncnn binary in path or configured)
run: build-server
	@echo "Running server..."
	$(SERVER_BIN)

## models: Download the Real-ESRGAN models
models:
	@echo "Downloading models..."
	./scripts/download-models.sh

## docker-build: Build the Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f deployments/docker/Dockerfile .

## docker-run: Run the service using Docker Compose
docker-run:
	@echo "Starting service with Docker Compose..."
	$(DOCKER_COMPOSE) up -d --build

## docker-stop: Stop the Docker Compose service
docker-stop:
	@echo "Stopping service..."
	$(DOCKER_COMPOSE) down

## docker-logs: View service logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## test: Run unit tests
test:
	@echo "Running unit tests..."
	go test -v ./internal/...

## test-integration: Run integration tests (requires server running on port 8089)
test-integration:
	@echo "Running integration tests..."
	./test/integration/test_api.sh

## git-init: Initialize git repository and commit initial files
git-init:
	@echo "Initializing git..."
	git init
	git add .
	git commit -m "Initial commit: Image Upscale Service (v$(VERSION))"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'