# ---- Config ----
IMAGE := ghcr.io/rancher/ci-image/go1.26
WORKDIR := /workspace

# Detect CI environment (common env var used by many CI systems)
CI ?= false

# Docker run wrapper (only used locally)
DOCKER_RUN = docker run --rm -i \
	-v $(PWD):$(WORKDIR) \
	-w $(WORKDIR) \
	$(IMAGE)

# Command runner:
# - In CI: run commands directly
# - Locally: run via Docker
ifeq ($(CI),true)
	RUN =
else
	RUN = $(DOCKER_RUN)
endif

# ---- Default ----
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  ci          - Run the ci scripts"
	@echo "  test        - Run the test script"
	@echo "  lint        - Run golangci-lint"
	@echo "  release     - Run goreleaser"

TARGETS := $(shell ls scripts)

.PHONY: $(TARGETS)
$(TARGETS):
	$(RUN) ./scripts/$@

.PHONY: validate
validate: validate-dirty ## Run validation checks.

.PHONY: validate-dirty
validate-dirty:
ifdef DIRTY
	@echo Git is dirty
	@git --no-pager status
	@git --no-pager diff
	@exit 1
endif


# ---- Local targets (use Docker) ----
.PHONY: lint
lint:
	$(RUN) golangci-lint run

.PHONY: release
release:
	$(RUN) goreleaser release --snapshot --clean
