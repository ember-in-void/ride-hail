SHELL := /bin/bash

.PHONY: all fmt vet build run-ride run-driver run-admin run-all

all: fmt vet build

fmt:
	@gofumpt -l -w .

vet:
	@go vet ./...

build:
	@go build -o ride-hail-system .

run-ride:
	@./ride-hail-system -service=ride

run-driver:
	@./ride-hail-system -service=driver

run-admin:
	@./ride-hail-system -service=admin

run-all:
	@./ride-hail-system -service=all