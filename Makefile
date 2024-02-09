MAKEFLAGS += --no-print-directory

.DEFAULT_GOAL := help

.PHONY: build build-all build-linux build-windows clean fmt help install install-system lint quality run test test-cover uninstall uninstall-system

BINARY_NAME := gobi

export BINARY_NAME

help: ## Show available targets
	@echo "gobi - Available targets"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*## "} {printf "  %-20s %s\n", $$1, $$2}'

build: ## Write bin/gobi for the current host
	@./.make/build.sh

build-linux: ## Cross-compile bin/gobi-linux-amd64 (GOARCH=arm64 optional via ARCH=)
	@./.make/build-cross.sh linux $(or $(ARCH),amd64)

build-windows: ## Cross-compile bin/gobi-windows-amd64.exe (GOARCH via ARCH=)
	@./.make/build-cross.sh windows $(or $(ARCH),amd64)

build-all: build-linux build-windows ## build-linux and build-windows (amd64)

run: ## Build if needed and run bin/gobi (args after --)
	@./.make/run.sh $(filter-out $@,$(MAKECMDGOALS))

install: ## Install to ~/.local/bin (requires prior make build)
	@./.make/install.sh

install-system: ## Install to /usr/local/bin (requires prior make build; sudo only for copy)
	@SYSTEM=1 ./.make/install.sh

uninstall: ## Remove from ~/.local/bin (sudo only if needed)
	@./.make/uninstall.sh

uninstall-system: ## Remove from /usr/local/bin (sudo only if needed)
	@SYSTEM=1 ./.make/uninstall.sh

clean: ## Remove bin/* (keep .gitkeep), coverage.out, and coverage.html
	@./.make/clean.sh

fmt: ## Format code
	@go fmt ./...

lint: ## Run linter
	@go vet ./...

quality: fmt lint ## Run all quality checks

test: ## Run tests
	@./.make/test.sh

test-cover: ## Write coverage.out and coverage.html at the git root
	@./.make/test-cover.sh

# Absorb extra goals after -- for `make run -- -e QUIT`
%:
	@:
