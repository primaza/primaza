# ServiceCatalog

A ServiceCatalog represents a group of RegisteredServices which are available in a Environment.

The ServiceCatalog can be consumed by Kubernetes applications that are unaware of the lifecycle of the service.
Events on RegisteredServices signal an update to the ServiceCatalog.
The ServiceCatalog can then be advertised to the application developers.

## Specification

The definition of a ServiceCatalog can be obtained directly from our [ServiceCatalog CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_servicecatalogs.yaml).
The specification contains a list of `ServiceCatalogService`.

The ServiceCatalogService contains the following fields:

- `name`: Name defines the name of the known service
- `serviceClassIdentity`: A set of key/value pairs that identify the service class.
  Examples of service class identity keys include type of service, and provider of service.
  This property is required.
- `serviceEndpointDefinitionKeys`: An array of keys that is required for connectivity.
  The values corresponding to each of these keys will be extracted from the service.
  This property is required.

## Use Cases

### Creation

When a ClusterEnvironment is created, Primaza ensures a ServiceCatalog exists for its environment.
The ServiceCatalog thus created is then been pushed to all the Application Namespaces of matching ClusterEnvironment where permission is granted.

### Deletion

ServiceCatalog gets deleted when the ClusterEnvironment is being deleted.
The Catalog will also be removed from the matching Application Namespaces.

### Update

When a RegisteredService is updated, the appropriate ServiceCatalog will get updated as well.
The updated ServiceCatalog will be made available to all the Application Namespaces of matching ClusterEnvironments by Primaza.

The ServiceCatalog for each ClusterEnvironment will get updated with a service change if either the service has no constraints or the service has a constraint that matches the environment tag of the cluster environment.
Matching means the environment is either included explicitly or it is not excluded.
For more details look at the [Constraints section](./registeredservice.md#constraints) in the RegisteredService page.
