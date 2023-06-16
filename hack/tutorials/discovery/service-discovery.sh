#!/bin/env sh

## Configure the shell
TENANT="primaza-mytenant"
TENANT_CLUSTER_CONTEXT="kind-main"
TARGET_CLUSTER_CONTEXT="kind-worker" # would be main in Single-Cluster and Single-Namespace scenarios
SERVICE_NAMESPACE="services"         # would be primaza-mytenant in Single-Namespace scenario

## Install the MondoDB Operator
helm repo add mongodb https://mongodb.github.io/helm-charts
helm repo update
helm install community-operator mongodb/community-operator \
    --namespace "$SERVICE_NAMESPACE" \
    --kube-context "$TARGET_CLUSTER_CONTEXT" \
    --create-namespace

## Create the MongoDB Database
cat << EOF | kubectl apply -f - \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$SERVICE_NAMESPACE"
apiVersion: mongodbcommunity.mongodb.com/v1
kind: MongoDBCommunity
metadata:
  name: mongodb
spec:
  members: 3
  type: ReplicaSet
  version: "6.0.5"
  security:
    authentication:
      modes: ["SCRAM"]
  users:
    - name: my-user
      db: admin
      passwordSecretRef: # a reference to the secret that will be used to generate the user's password
        name: mongodb-my-user-password
      roles:
        - name: clusterAdmin
          db: admin
        - name: userAdminAnyDatabase
          db: admin
      scramCredentialsSecretName: my-scram
  additionalMongodConfig:
    storage.wiredTiger.engineConfig.journalCompressor: zlib
---
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-my-user-password
type: Opaque
stringData:
  password: $(tr -cd '[:alnum:]' < /dev/urandom | fold -w16 | head -n1)
EOF

## Grant Primaza's Service Agent the right to look for MongoDB instances
cat << EOF | kubectl apply -f - \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$SERVICE_NAMESPACE"
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: primaza-svc-community-mongodb
rules:
- apiGroups:
  - mongodbcommunity.mongodb.com
  resources:
  - mongodbcommunity
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: primaza-svc-community-mongodb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: primaza-svc-community-mongodb
subjects:
- kind: ServiceAccount
  name: primaza-svc-agent
  namespace: $SERVICE_NAMESPACE
EOF

## Deploy the ServiceClass
cat << EOF | kubectl apply -f - \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$TENANT"
apiVersion: primaza.io/v1alpha1
kind: ServiceClass
metadata:
  name: mongodb
spec:
  serviceClassIdentity:
  - name: type
    value: database
  - name: operator
    value: community
  - name: engine
    value: mongodb
  resource:
    apiVersion: mongodbcommunity.mongodb.com/v1
    kind: MongoDBCommunity
    serviceEndpointDefinitionMappings:
      secretRefFields:
      - name: password
        secretName:
          jsonPath: .spec.users[0].passwordSecretRef.name
        secretKey:
          constant: password
      resourceFields:
      - name: username
        jsonPath: .spec.users[0].name
        secret: true
      - name: database
        jsonPath: .spec.users[0].db
        secret: false
      - name: host
        jsonPath: .metadata.name
        secret: false
EOF

## Wait for the RegisteredService to be created
until kubectl get registeredservices mongodb \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$TENANT" \
    --output yaml; \
        do sleep 10; done
