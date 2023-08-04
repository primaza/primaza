# ServiceClaim

A ServiceClaim represents a claim for a RegisteredService.

ServiceBindings may explicitly request an Application by Name or by LabelSelector.
LabelSelector can match more than one resource.

## Specification

The definition of a ServiceClaim can be obtained directly from our [ServiceClaim CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_serviceclaims.yaml).

The specification contains the following properties:

- `serviceClassIdentity`: A set of key/value pairs that identify the service class.
  Examples of service class identity keys include type of service, and provider of service.
  This property is required.
- `serviceEndpointDefinitionKeys`: An array of keys that is required for connectivity.
  The values corresponding to each of these keys will be extracted from the service.
  This property is required.
- `application`: Fields identifies application resources through kind, apiVersion, and label selector or name.
- `environmentTag`: A string representing one of the environment.
- `applicationClusterContext`: A combination of ClusterEnvironment resource name and namespace.
- `envs`: Envs allows projecting Service Endpoint Definition's data as Environment Variables in the Pod

The `environmentTag` and `applicationClusterContext` are mutually exclusive.

`application` field values are passed to the ServiceBinding resource.
The application's label selector and application name are mutually exclusive.

## Status

The Status of the ServiceClaim is also defined under the [ServiceClaimCRD](../../config/crd/bases/primaza.io_serviceclaims.yaml).
It contains a mandatory property to track the state.

The state could be either `Pending` or `Resolved` or `Invalid`.
If the state is `Resolved`, the RegisteredService claimed is tracked in the ServiceClaim status.
Indeed, the status field `registeredService` takes track of the `name` and `UID` of the claimed RegisteredService.

The spec of a ServiceClaim is not meant to be updated.
If a user updates the spec of a ServiceClaim then the status of ServiceClaim is updated as `Invalid` when Primaza Application Agent attempts to update the ServiceClaim on Primaza Control Plane.

There is an optional `claimID` field with a unique ID for the claim.

<!-- TODO: Add conditions description -->

## Use Cases

### Creation

When a ServiceClaim is created, Primaza should find a RegisteredService based on `ServiceClassIdentity` and `ServiceEndpointDefinitionKeys` and create Secret and ServiceBinding resources.
The EnvironmentVariables `envs` declared in the ServiceClaim are also added to the ServiceBinding specification.
The ServiceBinding resource will be marked as the owner for the secret.
Then it will update the state of ServiceClaim to `Resolved`.
The state of RegisteredService will be changed to `Claimed`.

If no match for RegisteredService is found, the state of ServiceClaim will be set to `Pending`.

### Deletion

When a ServiceClaim is deleted, Primaza will delete the Service Endpoint Definition Secret and the ServiceBinding.
As ServiceBinding is the owner of the Service Endpoint Definition Secret, deleting it ensures deletion of the secret too.
It also change the state of RegisteredService to `Available`.

### Update

When a ServiceClaim is updated, Primaza will update the Service Endpoint Definition Secret, the ServiceBinding and the ServiceClaim's state accordingly.
The state changes will happen similar to that of creation time.
