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
The Service Catalog thus created is then been pushed to all the application namespaces of the Cluster Environment where permission is granted.

### Deletion

Service Catalog gets deleted when the Cluster Environment is being deleted.
The Catalog will also be removed from the matching application namepsaces.

### Update

When a registered service is being updated the Service Catalog will get updated too. The updated Sevice Catalog will be made available to all the matching application namespaces of Cluster Environment by the primaza agent app controller.
