# ServiceClass

A Service Class is intended to define how a registered service can be automatically generated from a service.
Currently, this service takes the shape of a resource already existing within a cluster: the Service Class controller will recognize these services and create a corresponding registered service.

This resource exists on both primaza and worker environments.
The Primaza environment will push these resources to worker environments, where the service agent will capture the state of services within its namespace. 

## Specification

Within a Service Class, there are two required properties:
- `resource` defines the APIVersion and Kind of the resource to be transformed into a service mapping.
  It also contains a service endpoint definition mapping, which defines how binding information for a particular service may be obtained.
  This mapping contains the name of the key and information on how to obtain the value associated with the key.
  This information can be a jsonpath, with a `secret` flag to determine if this key should be put in the secret associated with the generated registered services.
  Alternatively, it can contain a reference to a secret within the namespace.
  In this case, the results of this lookup will always be stored in the generated secret.
- `serviceClassIdentity` defines a set of attributes that are sufficient to identify a Service Class.
  This field is copied to the generated registered services.

A Service Class also contains two optional properties, `constraints` and `healthCheck`.
Both of these fields correspond exactly to their identically-named properties within the Registered Service resource.
For more information on how to use these properties, refer to the [Registered Service documentation](./registeredservices.md)

## Status

Whenever a Service Class is created or updated, a connection test from the service environment to Primaza is performed.
The status of the Service Class will be updated to contain the results of this test underneath the condition type `Connection`.

## Use Cases

### Creation

When a Service Class is created, Primaza pushes it to Service Namespaces whose Cluster Environment satisfies its constraints.
Once created, the service agent will collect all the services corresponding to that Service Class and will push the Registered Services it generates back to Primaza.

### Deletion

When a Service Class is deleted, Primaza removes it from Service Namespaces whose Cluster Environment satisfies its constraints.
The Service Agent will identify the Registered Services that correspond to that Service Class and will delete them.

### Update

When a Service Class is updated, Primaza pushes it to Service Namespaces whose Cluster Environment satisfies its constraints.
The Service Agent will then collect the services corresponding to that Service Class and will create or update the Registered Services in Primaza.
