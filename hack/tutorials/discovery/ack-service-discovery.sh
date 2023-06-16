#!/bin/env sh

## Configure the shell
TENANT="primaza-mytenant"
TENANT_CLUSTER_CONTEXT="kind-main"
TARGET_CLUSTER_CONTEXT="kind-worker" # would be kind-main in Single-Cluster and Single-Namespace scenarios
SERVICE_NAMESPACE="services"         # would be primaza-mytenant in Single-Namespace scenario
AWS_REGION=$( aws configure get region )
AWS_RDS_RELEASE_VERSION=$( \
    curl -sL https://api.github.com/repos/aws-controllers-k8s/rds-controller/releases/latest | \
         grep '"tag_name":' | cut -d'"' -f4 | tr -d "v" )

## Install the ACK RDS operator
aws ecr-public get-login-password --region "us-east-1" | \
    helm registry login --username "AWS" --password-stdin "public.ecr.aws"

helm upgrade --install "ack-rds-controller" \
    "oci://public.ecr.aws/aws-controllers-k8s/rds-chart" \
    --namespace "$SERVICE_NAMESPACE" \
    --create-namespace \
    --kube-context "$TARGET_CLUSTER_CONTEXT" \
    --version="$AWS_RDS_RELEASE_VERSION" \
    --set=aws.region="$AWS_REGION" \
    --set=installScope=namespace

kubectl set env "deployment/ack-rds-controller-rds-chart" \
    AWS_ACCESS_KEY_ID="$( aws configure get aws_access_key_id )" \
    AWS_SECRET_ACCESS_KEY="$( aws configure get aws_secret_access_key )" \
    --namespace "$SERVICE_NAMESPACE" \
    --context "$TARGET_CLUSTER_CONTEXT"

## Create the MongoDB Database
cat << EOF | kubectl apply -f - \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$SERVICE_NAMESPACE"
apiVersion: rds.services.k8s.aws/v1alpha1
kind: DBInstance
metadata:
  name: rds
spec:
  allocatedStorage: 20
  dbInstanceClass: db.t3.micro
  dbInstanceIdentifier: primaza-ack-tutorial
  engine: postgres
  engineVersion: "14"
  masterUsername: "postgres"
  masterUserPassword:
    namespace: "$SERVICE_NAMESPACE"
    name: rds-password
    key: password
---
apiVersion: v1
kind: Secret
metadata:
  name: rds-password
stringData:
  password: $(tr -cd '[:alnum:]' < /dev/urandom | fold -w16 | head -n1)
EOF

## Grant Primaza's Service Agent the right to look for ACK DBInstances
cat << EOF | kubectl apply -f - \
    --context "$TARGET_CLUSTER_CONTEXT" \
    --namespace "$SERVICE_NAMESPACE"
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: primaza-svc-ack-rds
rules:
- apiGroups:
  - rds.services.k8s.aws
  resources:
  - dbinstances
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: primaza-svc-ack-rds
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: primaza-svc-ack-rds
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
  name: rds
spec:
  serviceClassIdentity:
  - name: type
    value: database
  - name: operator
    value: ack
  - name: provider
    value: aws
  resource:
    apiVersion: rds.services.k8s.aws/v1alpha1
    kind: DBInstance
    serviceEndpointDefinitionMappings:
      secretRefFields:
      - name: password
        secretName:
          jsonPath: .spec.masterUserPassword.name
        secretKey:
          jsonPath: .spec.masterUserPassword.key
      resourceFields:
      - name: username
        jsonPath: .spec.masterUsername
        secret: true
      - name: region
        jsonPath: .status.ackResourceMetadata.region
        secret: false
      - name: port
        jsonPath: .status.endpoint.port
        secret: false
      - name: host
        jsonPath: .status.endpoint.address
EOF

## Wait for the RegisteredService to be created
until kubectl get registeredservices rds \
    --context "$TENANT_CLUSTER_CONTEXT" \
    --namespace "$TENANT" \
    --output yaml
    do sleep 10; done
