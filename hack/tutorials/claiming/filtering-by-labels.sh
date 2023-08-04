#!/bin/env sh

## Configure the shell
TENANT="primaza-mytenant"
TENANT_CLUSTER_CONTEXT="kind-main"
TARGET_CLUSTER_CONTEXT="kind-worker" # it would be main in Single-Cluster and Single-Namespace scenarios
APPLICATION_NAMESPACE="applications" # it would be primaza-mytenant in Single-Namespace scenario

## Create a RegisteredService for the service to bind to our application
cat << EOF | kubectl apply -f - \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$TENANT"
apiVersion: primaza.io/v1alpha1
kind: RegisteredService
metadata:
  name: claim-by-labels
spec:
  serviceClassIdentity:
  - name: type
    value: dummy
  - name: scope
    value: claim-by-labels
  serviceEndpointDefinition:
  - name: url
    value: https://my-app-by-labels-service.dev
  - name: password
    value: SomeoneThinksImAPassword
EOF

## Create our target application
kubectl create deployment my-app-labels --image bash:latest \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$APPLICATION_NAMESPACE" \
    -- sleep infinity

## Create the ServiceClaim in Primaza's Control Plane
cat << EOF | kubectl apply -f - \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$TENANT"
apiVersion: primaza.io/v1alpha1
kind: ServiceClaim
metadata:
  name: my-app-dummy-by-labels
spec:
  serviceClassIdentity:
  - name: type
    value: dummy
  - name: scope
    value: claim-by-labels
  serviceEndpointDefinitionKeys:
  - url
  - password
  target:
    environmentTag: demo
  application:
    kind: Deployment
    apiVersion: apps/v1
    selector:
      matchLabels:
        app: my-app-labels
EOF

## Wait for Service Binding to be created and Resolved
until kubectl get servicebindings my-app-dummy-by-labels \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$APPLICATION_NAMESPACE"
do sleep 2; done

kubectl wait --for=condition=Bound servicebindings my-app-dummy-by-labels \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$APPLICATION_NAMESPACE"

## Read data injected in the workload
# shellcheck disable=SC2016  # mdbash: skip-line
until kubectl exec "pod/$(kubectl get pod \
        -l app=my-app-labels \
        -o jsonpath="{.items[0].metadata.name}" \
        --context "$TARGET_CLUSTER_CONTEXT" \
        --namespace "$APPLICATION_NAMESPACE" )" \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$APPLICATION_NAMESPACE" \
    -- bash -c '[ -d "/bindings/my-app-dummy-by-labels" ] && ( ls /bindings/my-app-dummy-by-labels | xargs -I@ bash -c '"'"'echo "$1: $(cat /bindings/my-app-dummy-by-labels/$1)"'"'"' -- @ )'
    do sleep 5; done
