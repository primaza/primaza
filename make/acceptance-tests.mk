##@ Acceptance Tests

TEST_ACCEPTANCE_OUTPUT_DIR ?= $(OUTPUT_DIR)/acceptance-tests
TEST_ACCEPTANCE_CLI ?= kubectl
TEST_ACCEPTANCE_PARALLEL ?= 4
TEST_ACCEPTANCE_TAGS ?=

ifdef TEST_ACCEPTANCE_TAGS
TEST_ACCEPTANCE_TAGS_ARG ?= --tags="~@disabled" --tags="$(TEST_ACCEPTANCE_TAGS)"
else
TEST_ACCEPTANCE_TAGS_ARG ?= --tags="~@disabled"
endif

ACCEPTANCE_TEST_TARGETS := test-acceptance test-acceptance-dr test-acceptance-x test-acceptance-wip test-acceptance-wip-x

$(ACCEPTANCE_TEST_TARGETS): ensure-agentsvc-image ensure-agentapp-image ensure-controller-image

PRIMAZA_CONTROLLER_IMAGE_REF ?= primaza-controller:latest
PRIMAZA_AGENTAPP_IMAGE_REF ?= agentapp:latest
PRIMAZA_AGENTSVC_IMAGE_REF ?= agentsvc:latest
export PRIMAZA_CONTROLLER_IMAGE_REF
export PRIMAZA_AGENTAPP_IMAGE_REF
export PRIMAZA_AGENTSVC_IMAGE_REF

.PHONY: ensure-controller-image
ensure-controller-image:
ifeq ($(origin PRIMAZA_CONTROLLER_IMAGE_REF), file)
	$(MAKE) primaza docker-build IMG=$(PRIMAZA_CONTROLLER_IMAGE_REF)
else
ifneq ($(origin PULL_IMAGES), undefined)
	docker pull $(PRIMAZA_CONTROLLER_IMAGE_REF)
endif
endif
	@echo "using $(PRIMAZA_CONTROLLER_IMAGE_REF) as primaza controller"

.PHONY: ensure-agentapp-image
ensure-agentapp-image:
ifeq ($(origin PRIMAZA_AGENTAPP_IMAGE_REF), file)
	$(MAKE) agentapp docker-build IMG=$(PRIMAZA_AGENTAPP_IMAGE_REF)
else
ifneq ($(origin PULL_IMAGES), undefined)
	docker pull $(PRIMAZA_AGENTAPP_IMAGE_REF)
endif
endif
	@echo "using $(PRIMAZA_AGENTAPP_IMAGE_REF) as application agent"

.PHONY: ensure-agentsvc-image
ensure-agentsvc-image:
ifeq ($(origin PRIMAZA_AGENTSVC_IMAGE_REF), file)
	$(MAKE) agentsvc docker-build IMG=$(PRIMAZA_AGENTSVC_IMAGE_REF)
else
ifneq ($(origin PULL_IMAGES), undefined)
	docker pull $(PRIMAZA_AGENTSVC_IMAGE_REF)
endif
endif
	@echo "using $(PRIMAZA_AGENTSVC_IMAGE_REF) as service agent"

.PHONY: setup-venv
setup-venv: ## Setup virtual environment
	python3 -m venv $(PYTHON_VENV_DIR)
	$(PYTHON_VENV_DIR)/bin/pip install --upgrade pip wheel setuptools

.PHONY: test-acceptance-setup
test-acceptance-setup: setup-venv ## Setup the environment for the acceptance tests
	$(PYTHON_VENV_DIR)/bin/pip install -q -r test/acceptance/features/requirements.txt

.PHONY: test-acceptance
test-acceptance: test-acceptance-setup ## Runs acceptance tests
	@(kind get clusters | grep primaza | xargs -I@ kind delete cluster --name @) || true
	echo "Running acceptance tests"
	$(PYTHON_VENV_DIR)/bin/behave --junit --junit-directory $(TEST_ACCEPTANCE_OUTPUT_DIR) --no-capture --no-capture-stderr $(TEST_ACCEPTANCE_TAGS_ARG) $(EXTRA_BEHAVE_ARGS) test/acceptance/features

.PHONY: test-acceptance-dr
test-acceptance-dr: test-acceptance-setup ## Runs acceptance tests
	echo "Running acceptance tests dry-run"
	$(PYTHON_VENV_DIR)/bin/behave --dry-run --junit --junit-directory $(TEST_ACCEPTANCE_OUTPUT_DIR) --no-capture --no-capture-stderr $(TEST_ACCEPTANCE_TAGS_ARG) $(EXTRA_BEHAVE_ARGS) test/acceptance/features

.PHONY: test-acceptance-x
test-acceptance-x: test-acceptance-setup kustomize controller-gen opm ## Runs acceptance tests in parallel
	@(kind get clusters | grep primaza | xargs -I@ kind delete cluster --name @) || true
	echo "Running acceptance tests in $(TEST_ACCEPTANCE_PARALLEL) parallel processes"
	FEATURES_PATH=test/acceptance/features $(PYTHON_VENV_DIR)/bin/behavex -o $(TEST_ACCEPTANCE_OUTPUT_DIR) --no-capture --no-capture-stderr $(TEST_ACCEPTANCE_TAGS_ARG) $(EXTRA_BEHAVE_ARGS) --parallel-processes $(TEST_ACCEPTANCE_PARALLEL) --stop

.PHONY: clean
clean: ## Removes temp directories
	-rm -rf ${V_FLAG} $(OUTPUT_DIR)
