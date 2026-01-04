# This project uses https://mise.jdx.dev/ as its task runner.
# The following Makefile is a wrapper around calls to mise
#
# Usage:
# make        - List all available tasks
# make [task] - Run a task

.DEFAULT_GOAL := default

ifeq (, $(shell command -v mise))
$(error "This project requires https://mise.jdx.dev to be installed.")
endif

TARGETS := $(if $(MAKECMDGOALS), $(subst :,\,$(MAKECMDGOALS)),$(.DEFAULT_GOAL))

.PHONY: $(TARGETS)

$(TARGETS): %:
	@if [ $@ = $(DEFAULT_GOALS) ]; then mise tasks; else mise run $@; fi
