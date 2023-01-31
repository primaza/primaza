KUSTOMIZE_ARGS=--load-restrictor LoadRestrictionsNone
KUSTOMIZE_OVERLAY=config/crd/overlays/agentsvc
NAMESPACE?=default

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build $(KUSTOMIZE_ARGS) $(KUSTOMIZE_OVERLAY) | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build $(KUSTOMIZE_ARGS) $(KUSTOMIZE_OVERLAY) | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/agents/svc && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/agents/svc | kubectl apply  -n $(NAMESPACE) -f -
	kubectl rollout status deploy/primaza-controller-agentsvc -n $(NAMESPACE) -w --timeout=120s

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/agents/svc | kubectl delete --ignore-not-found=$(ignore-not-found) -n $(NAMESPACE) -f -
