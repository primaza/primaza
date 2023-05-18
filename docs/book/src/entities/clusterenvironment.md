# ClusterEnvironment

A ClusterEnvironment represents a development environment on a kubernetes Cluster.
Examples of environments are 'app1-prod','app1-dev', or 'app1-uat'.

ClusterEnvironments contain connection information and the namespaces in which Primaza should operate.
ClusterEnvironments differentiate among Service and Application namespaces.
Application namespaces are the ones in which Primaza pushes the Application Agent that in turn binds applications to services.
Service namespaces are the ones in which Primaza pushes the Service Agent and that in turn performs service discovery.
Please refer to the [Architecture section](../architecture/agents.md) for more information about Agents and Primaza's architecture.

## Specification

The definition of ClusterEnvironments can be obtained directly from our [ClusterEnvironment CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_clusterenvironments.yaml).

The ClusterEnvironment's specification contains the following **required** properties:

- `clusterContextSecret`: contains the name of the secret that stores the kubeconfig that can be used to connect to the physical target cluster.
- `applicationNamespaces`: contains a list of namespaces where claiming and binding will happen.
   Applications to be bound to services will be looked for in those namespaces.
- `serviceNamespaces`: contains a list of namespaces where discovery will happen.
  Services that populate the Service Catalog will be looked for in those namespaces.

A ClusterEnvironment also defines the following **optional** properties:

- `contactInfo` Cluster Admin's contact information
- `description`: Description of the ClusterEnvironment

## Status

The ClusterEnvironment's status can have one of the following values:
- `Online`
- `Partial`
- `Offline`

An `Online` ClusterEnvironment is reachable by Primaza, whereas an `Offline` one is not reachable.

A `Partial` ClusterEnvironment is also reachable, but not configured properly.
This can happen if Primaza does not have the required permissions on this namespaces.
More details can be found in the ClusterEnvironment's status conditions.

<!-- TODO: Add conditions description -->

<!-- TODO(@baiju): Healtcheck section -->
<!-- ## Healthcheck -->

## Use Cases

### Creation

When a ClusterEnvironment is created, Primaza verifies the connection to the cluster.
If it can not connect to the target cluster, it logs an error and retries later.
Otherwise, it checks its permissions in application and service namespaces.
For each service and application namespace on which permissions are granted, Primaza pushes respectively the service or application agent.

ClusterEnvironment's State and Conditions are updated according to tests and agents' deployment results.

When a ClusterEnvironment is created, Primaza ensures a Service Catalog exists for its environment.
The Service Catalog thereby created are also pushed to the ClusterEnvironment application namespace where permissions are granted.

### Deletion

When a ClusterEnvironment is deleted, the permissions granted in Primaza's namespace to Service Accounts associated to namespace agents and agent deployments on target cluster's namespaces are removed.

### Update

As on [creation](#creation), Primaza verifies the connection to and its permissions into the target cluster.
Finally, it pushes agents in cluster's application and service namespaces.
As on [deletion](#deletion), if application or service namespaces are removed, Primaza deletes agent deployments and agents-granted permissions.

