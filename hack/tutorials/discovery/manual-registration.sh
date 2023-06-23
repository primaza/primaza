#!/bin/bash

## Configure the shell
TENANT="primaza-mytenant"
TENANT_CLUSTER_CONTEXT="kind-main"
SERVICE_NAMESPACE="services" # would be primaza-mytenant in Single-Namespace scenario

## Create a PostgreSQL Service
cat << EOF | kubectl apply -f - \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$SERVICE_NAMESPACE"
apiVersion: v1
kind: Secret
metadata:
  name: postgresql
stringData:
  database: postgresql-db
  username: postgresql-user
  password: postgresql-passwd
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgresql
  labels:
    app: postgresql
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgresql
  template:
    metadata:
      labels:
        app: postgresql
    spec:
      containers:
        - name: postgresql
          image: postgres:13
          imagePullPolicy: IfNotPresent
          env:
            - name: POSTGRES_DB_FILE
              value: /secrets/database
            - name: POSTGRES_PASSWORD_FILE
              value: /secrets/password
            - name: POSTGRES_USER_FILE
              value: /secrets/username
            - name: PGDATA
              value: /tmp/data
          volumeMounts:
            - name: postgresql
              mountPath: "/secrets"
              readOnly: true
          ports:
            - name: postgresql
              containerPort: 5432
      volumes:
        - name: postgresql
          secret:
            secretName: postgresql
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: postgresql
  name: postgresql
spec:
  ports:
    - port: 5432
      protocol: TCP
      targetPort: 5432
  selector:
    app: postgresql
EOF

## Create the RegisteredService and the Service Endpoint Definition Secret for the PostgreSQL Service in Primaza's Control Plane
cat << EOF | kubectl apply -f - \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$TENANT"
apiVersion: v1
kind: Secret
metadata:
  name: postgresql
stringData:
  username: postgresql-user
  password: postgresql-passwd
---
apiVersion: primaza.io/v1alpha1
kind: RegisteredService
metadata:
  name: postgres
spec:
  serviceClassIdentity:
  - name: type
    value: database
  - name: engine
    value: postgres
  - name: version
    value: "13"
  serviceEndpointDefinition:
  - name: database
    value: postgres
  - name: host
    value: postgresql.$SERVICE_NAMESPACE.svc.cluster.local
  - name: port
    value: "5432"
  - name: password
    valueFromSecret:
      name: postgresql
      key: password
  - name: username
    valueFromSecret:
      name: postgresql
      key: username
EOF

