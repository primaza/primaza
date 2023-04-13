KUSTOMIZE_ARGS=--load-restrictor LoadRestrictionsNone
KUSTOMIZE_OVERLAY=config/crd/overlays/agentapp
NAMESPACE?=default
CLUSTER_ENVIRONMENT?=default

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build $(KUSTOMIZE_ARGS) $(KUSTOMIZE_OVERLAY) | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build $(KUSTOMIZE_ARGS) $(KUSTOMIZE_OVERLAY) | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy-rbac
deploy-rbac:
	cd config/agents/app/rbac && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	$(KUSTOMIZE) build config/agents/app/rbac \
		| CLUSTER_ENVIRONMENT=$(CLUSTER_ENVIRONMENT) envsubst '$$CLUSTER_ENVIRONMENT' \
		| kubectl apply -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/agents/app && \
		$(KUSTOMIZE) edit set image agentapp=${IMG} && \
		$(KUSTOMIZE) edit set namespace $(NAMESPACE)
	$(KUSTOMIZE) build $(KUSTOMIZE_ARGS) config/agents/app \
		| CLUSTER_ENVIRONMENT=$(CLUSTER_ENVIRONMENT) envsubst '$$CLUSTER_ENVIRONMENT' \
		| kubectl apply -f -
	kubectl rollout status deploy/pmz-app-$(CLUSTER_ENVIRONMENT) -n $(NAMESPACE) -w --timeout=120s

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	cd config/agents/app && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	$(KUSTOMIZE) build $(KUSTOMIZE_ARGS) config/agents/app \
		| CLUSTER_ENVIRONMENT=$(CLUSTER_ENVIRONMENT) envsubst '$$CLUSTER_ENVIRONMENT' \
		| kubectl delete --ignore-not-found=$(ignore-not-found) -f -
