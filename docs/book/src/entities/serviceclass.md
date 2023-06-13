# ServiceClass

A Service Class is intended to define how a registered service can be automatically generated from a service.
Currently, this service takes the shape of a resource already existing within a cluster: the Service Class controller will recognize these services and create a corresponding registered service.

This resource exists on both primaza and worker environments.
The Primaza environment will push these resources to worker environments, where the service agent will capture the state of services within its namespace. 

## Specification

The definition of ServiceClasses can be obtained directly from our [ServiceClass CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_serviceclasses.yaml).

Within a Service Class, there are two required properties:
- `resource` defines the APIVersion and Kind of the resources to look for.
  It also contains a list of Service Endpoint Definition Mappings, which define how binding information for a particular service may be obtained.
  For more details, have a look at the [resource field section](#resource-field).
- `serviceClassIdentity` defines a set of attributes that are sufficient to identify a Service Class.
  This field is copied to the generated registered services.

A Service Class also contains two optional properties, `constraints` and `healthCheck`.
Both of these fields correspond exactly to their identically-named properties within the Registered Service resource.
For more information on how to use these properties, refer to the [Registered Service documentation](./registeredservices.md)

### `resource` field

The `resource`'s ServiceClass field contains all the information needed for identifying the resources it refers to, i.e. `apiVersion` and `kind`.
Also, it contains the rules for extracting the Service Endpoint Definition secret data, i.e. `serviceEndpointDefinitionMappings`.

Mapping rules apply to the resource specification (`resourceFields`) or to a secret (`secretRefFields`).

To extract data from the resource, you define an entry in the `resourceFields` list.
Each entry contains the following properties:
* `name`: is used as the key in the Registered Service's Service Endpoint Definition
* `jsonPath`: a JSONPath rule to extract the value for the current Service Endpoint Definition key
* `secret`: declares whether this field should be stored in a secret of if it can be embedded in the Registered Service specification

To extract data from a secret, you define an entry in the `secretRefFields` list.
Data extracted from a secret is always stored in a new secret.
Each entry contains the following properties:
* `name`: is used as the key in the Registered Service's Service Endpoint Definition
* `secretName`: represents the name of the secret to look for.
  It that has the following two mutually exclusive sub-properties:
    * `jsonPath`: a JSONPath rule to extract the name of the secret from the resource specification
    * `constant`: a constant value for the secret name
* `secretKey`: represents the secret's key to use.
  It has the following two mutually exclusive sub-properties:
    * `jsonPath`: a JSONPath rule to extract the key of the secret from the resource specification
    * `constant`: a constant value for the secret name

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
