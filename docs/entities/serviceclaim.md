# ServiceClaim

A Service Claim represents a claim for a Registered Service.

## Specification

The definition of a ServiceClaim can be obtained directly from our [ServiceClaim
CRD](../../config/crd/bases/primaza.io_serviceclaims.yaml). The specification
contains two required properties:

- ServiceClassIdentity: A set of key/value pairs that identify the service
  class. Examples of service class identity keys include type of service, and
  provider of service. This property is required.
- ServiceEndpointDefinitionKeys: An array of keys that is required for
  connectivity. The values corresponding to each of these keys will be extracted
  from the service. This property is required.

There are other three optional fields. The EnvironmentTag and
ApplicationClusterContext are mutually exclusive:

- Application: Fields indentifies application resources through kind, apiVersion and label selector.
- EnvironmentTag: A string representing one of the environment.
- ApplicationClusterContext: A combination of ClusterEnvironment resource name and namespace.

## Status

The Status of the ServiceClaim is also defined under the [ServiceClaim
CRD](../../config/crd/bases/primaza.io_serviceclaims.yaml). It contains a
mandatory property to track the state. The state could be either `Pending` or
`Resolved`. If the state is `Resolved`, there should be Secret and Service
Binding resources created. And there is another mandatory field,
`registeredService` that points to the Registered Service.

There is an optional `claimID` field with a unique ID for the claim.

## Use Cases

### Creation

When a Service Claim is created, Primaza should find a Registered Service based on Service Class Identity and Service Endpoint Definition Keys and create Secret and Service Binding resources. The Service Binding resource will be marked as the owner for the secret. Then it will update the state of Service Claim to `Resolved`.  The state of Registered Service will be changed to `Claimed`. If no match for Registered Service is found, the state of Service Claim will be set to `Pending`.

### Deletion

When a Service Claim is deleted, Primaza will delete the Service Endpoint Definition Secret and the Service Binding. As Service Binding is the owner of the Service Endpoint Definition Secret, deleting it ensures deletion of the secret too. It also change the state of Registered Service to `Available`.

### Update

When a Service Claim is updated, Primaza will update the Service Endpoint Definition Secret, the Service Binding and the Service Claim's state accordingly.  The state changes will happen similar to that of creation time.
