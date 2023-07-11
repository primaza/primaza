##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(PROJECT_DIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
PRIMAZACTL ?= $(LOCALBIN)/primazactl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
YQ ?= $(LOCALBIN)/yq

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.7
CONTROLLER_TOOLS_VERSION ?= v0.11.3
CERTMANAGER_VERSION ?= v1.11.1
YQ_VERSION ?= v4
PRIMAZACTL_VERSION ?= latest

.PHONY: primazactl
primazactl: $(PRIMAZACTL) ## Download primazactl locally if necessary.
$(PRIMAZACTL): $(LOCALBIN)
	test -s $(PRIMAZACTL) || { curl -sSL https://github.com/primaza/primazactl/releases/download/$(PRIMAZACTL_VERSION)/primazactl -o $(PRIMAZACTL) && chmod +x $(PRIMAZACTL); }

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) $(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) $(GO) install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl --retry 10 -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

.PHONY: yq
yq: $(YQ) ## Download envtest-setup locally if necessary.
$(YQ): $(LOCALBIN)
	test -s $(YQ) || GOBIN=$(LOCALBIN) $(GO) install github.com/mikefarah/yq/v4@$(YQ_VERSION)

$(OUTPUT_DIR)/cert-manager-$(CERTMANAGER_VERSION).yaml:
	curl -Lo $(OUTPUT_DIR)/cert-manager-$(CERTMANAGER_VERSION).yaml https://github.com/cert-manager/cert-manager/releases/download/$(CERTMANAGER_VERSION)/cert-manager.yaml

.PHONY: deploy-cert-manager
deploy-cert-manager: $(OUTPUT_DIR)/cert-manager-$(CERTMANAGER_VERSION).yaml
	kubectl apply -f $(OUTPUT_DIR)/cert-manager-$(CERTMANAGER_VERSION).yaml
	kubectl rollout status -n cert-manager deploy/cert-manager-webhook -w --timeout=120s
