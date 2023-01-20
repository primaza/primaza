#!/bin/env sh

main_profile=primaza-main
worker_profile=primaza-worker
namespace=primaza-system

CDIR=$(dirname "$(readlink -f "$0")")
PROJECT_DIR=$(realpath "$CDIR/../../../")
TMP_DIR="$CDIR/tmp"

set -e

echo ""
echo "Cleaning up..."
minikube --profile $worker_profile delete
minikube --profile $main_profile delete

export KUBECONFIG="$TMP_DIR"/kubeconfig
mkdir -p "$TMP_DIR"

echo ""
echo "Starting main cluster for Primaza"
minikube --profile $main_profile start
(
    cd "$PROJECT_DIR"
    make docker-build && minikube --profile $main_profile image load controller:latest
    make install deploy
)

echo ""
echo "Starting worker cluster for Primaza"
KUBECONFIG="$TMP_DIR/worker-config" minikube --profile $worker_profile start --kubernetes-version 1.25.2

# connect the clusters networks
docker network connect $worker_profile $main_profile
docker network connect $main_profile $worker_profile

echo ""
echo "Configure primaza user into the worker cluster"
DIR="$TMP_DIR" KUBECONFIG="$TMP_DIR/worker-config" "$PROJECT_DIR"/hack/configure-worker-cluster/configure-worker.sh

echo ""
echo "Create the cluster environment in Primaza main"
kubectl config use-context $main_profile
kubectl apply -f "$TMP_DIR/secret-config.yaml"
kubectl apply -f "$CDIR/clusterenvironment.yaml"
kubectl rollout status deployment -n $namespace primaza-controller-manager

kubectl wait \
    clusterenvironments.primaza.io \
    clusterenvironment-primaza-worker \
    --for=jsonpath='{.status.state}'=Online \
    --timeout=30s

# # MANUAL: look in logs for '"version": "v1.25.2"'
# kubectl logs -f -n $namespace deployments/primaza-controller-manager

