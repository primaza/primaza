##@ Linters

GOLANGCI_LINT=$(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.50.1

YAMLLINT_VERSION ?= 1.28.0

SHELLCHECK=$(LOCALBIN)/shellcheck
SHELLCHECK_VERSION ?= v0.9.0

.PHONY: lint
lint: setup-venv lint-go lint-yaml lint-python lint-feature-files lint-conflicts lint-shell ## Runs all linters

YAML_FILES := $(shell find . -path ./vendor -prune -o -path ./config -prune -o -path ./test/performance -prune -o -type f -regex ".*\.y[a]ml" -print)
.PHONY: lint-yaml
lint-yaml: setup-venv ${YAML_FILES} ## Checks all yaml files
	$(Q)$(PYTHON_VENV_DIR)/bin/pip install yamllint==$(YAMLLINT_VERSION)
	$(Q)$(PYTHON_VENV_DIR)/bin/yamllint -c .yamllint $(YAML_FILES)

GO_LINT_CMD = GOFLAGS="$(GOFLAGS)" GOGC=30 GOCACHE=$(GOCACHE) $(GOLANGCI_LINT) run --concurrency=1 --verbose --deadline=30m --disable-all --enable

.PHONY: lint-go
lint-go: $(GOLANGCI_LINT) fmt vet ## Checks Go code
	$(GO_LINT_CMD) gosimple
	$(GO_LINT_CMD) staticcheck
	$(GO_LINT_CMD) errcheck
	$(GO_LINT_CMD) govet
	$(GO_LINT_CMD) ineffassign
	$(GO_LINT_CMD) typecheck
	$(GO_LINT_CMD) unused

$(GOLANGCI_LINT):
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)

.PHONY: lint-python
lint-python: setup-venv ## Check python code
	PYTHON_VENV_DIR=$(PYTHON_VENV_DIR) $(HACK_DIR)/check-python/lint-python-code.sh

.PHONY: lint-feature-files
lint-feature-files: ## Check acceptance tests' feature files
	$(HACK_DIR)/check-feature-files.sh

.PHONY: lint-conflicts
lint-conflicts: ## Check for presence of conflict notes in source file
	$(HACK_DIR)/check-conflicts.sh

.PHONY: shellcheck
shellcheck: $(SHELLCHECK) ## Download shellcheck locally if necessary.
$(SHELLCHECK): $(OUTPUT_DIR) 
ifeq (,$(wildcard $(SHELLCHECK)))
ifeq (,$(shell which shellcheck 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(SHELLCHECK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH | sed -e 's,amd64,x86_64,g') && \
	curl -Lo $(OUTPUT_DIR)/shellcheck.tar.xz https://github.com/koalaman/shellcheck/releases/download/$(SHELLCHECK_VERSION)/shellcheck-$(SHELLCHECK_VERSION).$${OS}.$${ARCH}.tar.xz ;\
	tar --directory $(OUTPUT_DIR) -xvf $(OUTPUT_DIR)/shellcheck.tar.xz ;\
	find $(OUTPUT_DIR) -name shellcheck -exec cp {} $(SHELLCHECK) \; ;\
	chmod +x $(SHELLCHECK) ;\
	}
else
SHELLCHECK = $(shell which shellcheck)
endif
endif

.PHONY: lint-shell
lint-shell: $(SHELLCHECK) ## Check shell scripts
	find . -name vendor -prune -o -name '*.sh' -print | xargs $(SHELLCHECK) -x

.PHONY: lint-shell-fix
lint-shell-fix: $(SHELLCHECK)
	find * -name vendor -prune -o -name '*.sh' -type f -print | xargs -I@ sh -c "$(SHELLCHECK) -f diff @ | git apply"
