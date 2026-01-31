# Image Upscale Service
# Wrapper around Taskfile.yml - Please use 'task' directly if possible.

.PHONY: help build build-server build-client clean run models docker-build docker-run docker-stop docker-logs test test-integration

# Default target
help:
@echo "delegating to task..."
@task --list

build:
@task build

build-server:
@task build-server

build-client:
@task build-client

clean:
@task clean

run:
@task run

models:
@task models

docker-build:
@task docker-build

docker-run:
@task docker-run

docker-stop:
@task docker-stop

docker-logs:
@task docker-logs

test:
@task test

test-integration:
@task test-integration
