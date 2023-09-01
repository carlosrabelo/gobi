MAKEFLAGS += --no-print-directory

.DEFAULT_GOAL := help

.PHONY: build clean fmt help lint quality test

help: ## Show available targets
	@echo "gobi - Available targets"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*## "} {printf "  %-15s %s\n", $$1, $$2}'

build: ## Compile packages (does not write bin/gobi)
	@./.make/build.sh

clean: ## Remove bin/* (keep .gitkeep), coverage.out, and coverage.html
	@./.make/clean.sh

fmt: ## Format Go sources under gobi/
	@./.make/fmt.sh

lint: ## Run go vet
	@./.make/lint.sh

quality: ## Format, lint, and test
	@./.make/quality.sh

test: ## Run tests
	@./.make/test.sh
