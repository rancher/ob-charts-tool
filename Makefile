TARGETS := $(shell ls scripts)

$(TARGETS):
	./scripts/$@

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)

validate: validate-dirty ## Run validation checks.

validate-dirty:
ifdef DIRTY
	@echo Git is dirty
	@git --no-pager status
	@git --no-pager diff
	@exit 1
endif
