#!/bin/env bash

## Create the Main Cluster
kind create cluster --name main

## Install the Cert-Manager on Main Cluster
kubectl apply \
    -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml \
    --kubeconfig <(kind get kubeconfig --name main)
kubectl rollout status -n cert-manager deploy/cert-manager-webhook -w --timeout=120s \
    --kubeconfig <(kind get kubeconfig --name main)
sleep 60  # mdbash: skip-line

## Create a Primaza Tenant
primazactl create tenant primaza-mytenant \
    --version latest \
    --context kind-main

## Join the Main cluster
ip=$(docker container inspect main-control-plane --format '{{.NetworkSettings.Networks.kind.IPAddress}}')
kind get kubeconfig --name main | \
    sed 's/server: .*$/server: https:\/\/'"$ip"':6443/g' > /tmp/kc-primaza-single-setup
primazactl join cluster \
    --version latest \
    --tenant primaza-mytenant \
    --cluster-environment main \
    --environment demo \
    --context kind-main \
    --tenant-context kind-main \
    --kubeconfig /tmp/kc-primaza-single-setup

## Create an Application Namespace named "applications" in the Main Cluster
primazactl create application-namespace applications \
    --version latest \
    --tenant primaza-mytenant \
    --cluster-environment main \
    --context kind-main \
    --tenant-context kind-main

## Create a Service Namespace named "services" in the Main Cluster
primazactl create service-namespace services \
    --version latest \
    --tenant primaza-mytenant \
    --cluster-environment main \
    --context kind-main \
    --tenant-context kind-main
