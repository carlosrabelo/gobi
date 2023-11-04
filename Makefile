MAKEFLAGS += --no-print-directory

.DEFAULT_GOAL := help

.PHONY: build clean fmt help lint quality test test-cover run install install-system uninstall uninstall-system

help: ## Show available targets
	@echo "gobi - Available targets"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*## "} {printf "  %-15s %s\n", $$1, $$2}'

build: ## Write bin/gobi
	@./.make/build.sh

run: ## Build if needed and run bin/gobi (args after --)
	@./.make/run.sh $(filter-out $@,$(MAKECMDGOALS))

install: ## Copy bin/gobi to ~/.local/bin (no compile)
	@./.make/install.sh

install-system: ## Copy bin/gobi to /usr/local/bin (no compile)
	@./.make/install-system.sh

uninstall: ## Remove ~/.local/bin/gobi
	@./.make/uninstall.sh

uninstall-system: ## Remove /usr/local/bin/gobi
	@./.make/uninstall-system.sh

clean: ## Remove bin/* (keep .gitkeep), coverage.out, and coverage.html
	@./.make/clean.sh

fmt: ## Format Go sources under gobi/
	@./.make/fmt.sh

lint: ## Run go vet
	@./.make/lint.sh

quality: ## Format check, lint, and test
	@./.make/quality.sh

test: ## Run tests
	@./.make/test.sh

test-cover: ## Write coverage.out and coverage.html at the git root
	@./.make/test-cover.sh

# Absorb extra goals after -- for `make run -- -e QUIT`
%:
	@:
