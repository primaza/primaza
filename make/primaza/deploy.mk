##@ Deployment
NAMESPACE ?= primaza-system

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image primaza-controller=${IMG}
	cd config/manager && $(KUSTOMIZE) edit set health-check-interval ${HEALTH_CHECK_INTERVAL}
	cd config/default && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	kubectl rollout status -n $(NAMESPACE) deploy/primaza-controller-manager -w --timeout=120s

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	cd config/default && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
