##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="$(HACK_DIR)/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	$(GO) vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GO) test ./... -coverprofile cover.out

.PHONY: test-acceptance-wip
test-acceptance-wip: test-acceptance-setup ## Runs acceptance tests for WIP tagged scenarios
	@(kind get clusters | grep primaza | xargs -I@ kind delete cluster --name @) || true
	echo "Running work in progress acceptance tests"
	$(PYTHON_VENV_DIR)/bin/behave --junit --junit-directory $(TEST_ACCEPTANCE_OUTPUT_DIR) --no-capture --no-capture-stderr $(TEST_ACCEPTANCE_TAGS_ARG) $(EXTRA_BEHAVE_ARGS) --wip --stop test/acceptance/features

.PHONY: test-acceptance-wip-x
test-acceptance-wip-x: test-acceptance-setup ## Runs acceptance tests for WIP tagged scenarios
	@(kind get clusters | grep primaza | xargs -I@ kind delete cluster --name @) || true
	echo "Running work in progress acceptance tests in parallel"
	FEATURES_PATH=test/acceptance/features $(PYTHON_VENV_DIR)/bin/behavex -o $(TEST_ACCEPTANCE_OUTPUT_DIR) --no-capture --no-capture-stderr $(TEST_ACCEPTANCE_TAGS_ARG) $(EXTRA_BEHAVE_ARGS) -t="@wip" --stop --parallel-processes $(TEST_ACCEPTANCE_PARALLEL)

