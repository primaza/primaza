# ClusterEnvironment

A Cluster Environment represents a development environment on a kubernetes Cluster.
Examples of environments are 'app1-prod','app1-dev', or 'app1-uat'.

## Specification

Cluster Environments contain connection information and the namespaces in which Primaza should operate.

Connection information are stored in a Secret referred by the field `clusterContextSecret`.
The secret contains a valid kubeconfig that can be used to connect to the physical target cluster.

The field `applicationNamespaces` contains a list of namespaces where claiming and binding will happen.
Applications to be bound to services will be looked for in those namespaces.

The field `serviceNamespaces` contains a list of namespaces where discovery will happen.
Services that populate the Service Catalog will be looked for in those namespaces.

```yaml
spec:
  description: ClusterEnvironmentSpec defines the desired state of ClusterEnvironment
  properties:
    applicationNamespaces:
      description: Namespaces in target cluster where applications are deployed
      type: string
    clusterContextSecret:
      description: Name of the Secret where connection (kubeconfig) information
        to target cluster is stored
      type: string
    contactInfo:
      description: Cluster Admin's contact information
      type: string
    description:
      description: Description of the ClusterEnvironment
      type: string
    environmentName:
      description: The environment associated to the ClusterEnvironment
        instance
      type: string
    labels:
     description: Labels
      items:
        type: string
      type: array
    name:
      description: The name of the ClusterEnvironment
      type: string
    serviceNamespaces:
      description: Namespaces in target cluster where services are discovered
      type: string
  required:
  - clusterContextSecret
  - environmentName
  - name
```

## Status

A Cluster Environment can be `Online` or `Offline`.
An `Online` Cluster Environment is reachable by Primaza, whereas an `Offline` one is not reachable.

```yaml
status:
  description: ClusterEnvironmentStatus defines the observed state of ClusterEnvironment
  properties:
    state:
      default: Offline
      description: The State of the cluster environment
      enum:
      - Online
      - Offline
      type: string
  required:
  - state
```

## Use Cases

### Creation

When a Cluster Environment is created, Primaza must verify the connection and update the field `status.state` accordingly.

### Deletion

No operations should be performed.

### Update

When a Cluster Environment is updated, Primaza must verify the connection and update the field `status.state` accordingly.

