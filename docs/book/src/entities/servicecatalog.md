# ServiceCatalog

A Service Catalog represents a group of Registered Services which are available in a Environment.

## Specification

The definition of a ServiceCatalog can be obtained directly from our [ServiceCatalog CRD](../../config/crd/bases/primaza.io_servicecatalogs.yaml).
The specification contains a list of `ServiceCatalogService`.
The Service Catalog Service contains the following fields:

- Name: Name defines the name of the known service
- ServiceClassIdentity: A set of key/value pairs that identify the service
  class. Examples of service class identity keys include type of service, and
  provider of service. This property is required.
- ServiceEndpointDefinitionKeys: An array of keys that is required for
  connectivity. The values corresponding to each of these keys will be extracted
  from the service. This property is required.

## Use Cases

### Creation

When a Cluster Environment is created, Primaza ensures a Service Catalog exists for its environment.
The Service Catalog thus created is then been pushed to all the application namespaces of matching Cluster Environment where permission is granted.

### Deletion

Service Catalog gets deleted when the Cluster Environment is being deleted.
The Catalog will also be removed from the matching application namepsaces.

### Update

When a registered service is updated, the appropriate service catalog will get updated as well.
The updated service catalog will be made available to all the application namespaces of matching cluster environment by Primaza.
The `constraints` section of a registered service defines a list of environments.
The environment list can contain explicit environments that are to include the service, or it can also contain a list of environments that are to exclude the service.
The way to depict exclusion is by adding the `!` prefix to the environment, i.e `!prod`.
Please note that exclusions have precedence over inclusions.
For example, if the list contains `!prod` but also includes `dev`, then `dev` is considered to be in the `!prod` set of environments and therefore redundant.
If there is a third environment stage, then `!prod` would include both `stage` and `dev` even if they are not defined in the list explicitly.
The service catalogs for each cluster environment will get updated with a service change if either the service has no constraints or the service has a constraint that matches the environment tag of the cluster environment.
Matching means the environment is either included explicitly or it is not excluded. 
