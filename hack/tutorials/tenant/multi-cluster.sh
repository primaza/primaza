#!/bin/env bash

## Create the Main Cluster
kind create cluster --name main

## Install the Cert-Manager on Main Cluster
kubectl apply \
    -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml \
    --kubeconfig <(kind get kubeconfig --name main)
kubectl rollout status -n cert-manager deploy/cert-manager-webhook -w --timeout=120s \
    --kubeconfig <(kind get kubeconfig --name main)

## Create the Worker Cluster
kind create cluster --name worker

## Install the Cert-Manager on Worker Cluster
kubectl apply \
    -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml \
    --kubeconfig <(kind get kubeconfig --name worker)
kubectl rollout status -n cert-manager deploy/cert-manager-webhook -w --timeout=120s \
    --kubeconfig <(kind get kubeconfig --name worker)
sleep 30 # mdbash: skip-line

## Create a Primaza Tenant
primazactl create tenant primaza-mytenant \
    --version latest \
    --context kind-main

## Join the Worker cluster
primazactl join cluster \
    --version latest \
    --tenant primaza-mytenant \
    --cluster-environment worker \
    --environment demo \
    --context kind-worker \
    --tenant-context kind-main

## Create an Application Namespace named "applications" in the Worker Cluster
primazactl create application-namespace applications \
    --version latest \
    --tenant primaza-mytenant \
    --cluster-environment worker \
    --context kind-worker \
    --tenant-context kind-main

## Create a Service Namespace named "services" in the Worker Cluster
primazactl create service-namespace services \
    --version latest \
    --tenant primaza-mytenant \
    --cluster-environment worker \
    --context kind-worker \
    --tenant-context kind-main
