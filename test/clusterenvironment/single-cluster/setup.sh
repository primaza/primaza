#!/bin/env sh

profile=primaza-main
namespace=primaza-system

CDIR=$(dirname "$(readlink -f "$0")")
PROJECT_DIR=$(realpath "$CDIR/../../../")
TMP_DIR="$CDIR/tmp"

set -e

echo ""
echo "Cleaning up..."
minikube --profile $profile delete

echo ""
echo "Starting main cluster for Primaza"
export KUBECONFIG="$TMP_DIR/kubeconfig"
minikube --profile $profile start
(
    cd "$PROJECT_DIR"
    make docker-build && minikube --profile $profile image load controller:latest
    make install deploy
)

echo ""
echo "Configure primaza user into the main cluster"
DIR="$TMP_DIR" KUBECONFIG=$KUBECONFIG "$PROJECT_DIR/hack/configure-worker-cluster/configure-worker.sh"

echo ""
echo "Create the cluster environment in Primaza main"
kubectl apply -f "$CDIR/clusterenvironment.yaml"
kubectl apply -f "$TMP_DIR/secret-config.yaml"
kubectl rollout status deployment -n $namespace primaza-controller-manager

kubectl wait \
    clusterenvironments.primaza.io \
    clusterenvironment-primaza-worker \
    --for=jsonpath='{.status.state}'=Online \
    --timeout=30s
