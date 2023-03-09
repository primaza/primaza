# ClusterEnvironment

A Cluster Environment represents a development environment on a kubernetes Cluster.
Examples of environments are 'app1-prod','app1-dev', or 'app1-uat'.

Cluster Environments contain connection information and the namespaces in which Primaza should operate.
Cluster Environments differentiate among Service and Application namespaces.
Application namespaces are the ones in which Primaza pushes the Application Agent that in turn binds applications to services.
Service namespaces are the ones in which Primaza pushes the Service Agent and that in turn performs service discovery.
Please refer to the [Architecture section](../architecture/agents.md) for more information about Agents and Primaza's architecture.

## Specification

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

A Cluster Environment can be `Online`, `Partial, or `Offline`.
An `Online` Cluster Environment is reachable by Primaza, whereas an `Offline` one is not reachable.
A `Partial` Cluster Environment is also reachable, but not configured properly.
This can happen if Primaza does not have the required permissions on this namespaces.
More details can be found in the Cluster Environment's status conditions.

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
      - Partial
      type: string
  required:
  - state
```

## Use Cases

### Creation

When a Cluster Environment is created, Primaza verifies the connection to the cluster.
If it can not connect to the target cluster, it logs an error and retries later.
Otherwise, it checks its permissions in application and service namespaces.
For each service and application namespace on which permissions are granted, Primaza pushes respectively the service or application agent.

Cluster Environment's State and Conditions are updated according to tests and agents' deployment results.

### Deletion

When a Cluster Environment is deleted, the permissions granted in Primaza's namespace to Users associated to namespace agents and agent deployments on target cluster's namespaces are removed.

### Update

As on [creation](#creation), Primaza verifies the connection to and its permissions into the target cluster. Finally, it pushes agents in cluster's application and service namespaces.
As on [deletion](#deletion), if application or service namespaces are removed, Primaza deletes agent deployments and agents-granted permissions.

