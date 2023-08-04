# ServiceBinding

A ServiceBinding represents the secrets along with the Applications which are to bound with it.
ServiceBindings may explicitly request an Application by Name or by LabelSelector.
LabelSelector can match more than one resource.

ServiceBinding resource is being created by the `ServiceClaim` controller.

ServiceBinding is being performed by [Primaza Application Agent](../architecture/agents.md), and the agent should be deployed in the Application namespace where the workloads are to be bound.
When a ServiceBinding is created (or updated) into an Application namespace, the Application Agent gets the data from the secrets and project them into applications specified in the ServiceBinding instance.

Currently the secret data is being projected as volume mounts.
`SERVICE_BINDING_ROOT` points to the environment variable in the container which is used as the volume mount path.
In the absence of this environment variable, `/bindings` is used as the volume mount path.

Please refer to https://github.com/servicebinding/spec#reconciler-implementation for more information.

## Specification

The definition of ServiceBindings can be obtained directly from our [ServiceBinding CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_servicebindings.yaml).

The ServiceBinding's specification contains the following **required** properties:

- `serviceEndpointDefinitionSecret`: ServiceEndpointDefinitionSecret is the name of the secret to project into the application.
  This property is required.
- `application`: Application resource to inject the binding info.
  It could be any process running within a container.
  A ServiceBinding **MAY** define the application reference by name or by label selector.
  Name and label selector are mutually exclusive.

The ServiceBinding's specification also contains the following **optional** property:
- `envs`: Envs declares environment variables based on the        ServiceEndpointDefinitionSecret to be projected into the application

## Status

The ServiceBinding's status contains the properties `state` and `conditions`.

The state of the service binding can be `Malformed` or `Ready`.
The default value of service binding state is `Malformed`.

The `conditions` list of the service binding contains the following properties:
- `Type`: The service binding condition type is `Bound` or `NotBound`.
    - `Bound` means that the secret is projected into the application.
    - `NotBound` denotes that the secret is not projected into the application.
       This can only occur if the secret is not found in the application namespace.
- `Message`: This contains the error logs for the service binding resources.
  This value will be an empty string if successful.
- `Status`: Status of service binding can be `True` or `False`.
- `Reason`: The reason has values defined as `NoMatchingWorkloads`, `ErrorFetchSecret`, `Successful` and `Binding Failure`
- `Connections`: The list of workloads the service is bound to

## Use Cases

### Creation

When a ServiceBinding is created, the secret referenced by the ServiceBinding itself will be projected into all the matching applications.
The EnvironmentVariables `envs` declared in the specification will also be projected to the matching application pods.
Matching applications are calculated as defined at in the section [Specification](#specification)

### Deletion

If the ServiceBinding is deleted the secret projection from the workloads is removed too.
In case the secret referenced in the ServiceBinding resource is deleted, the projection is removed from the workloads and the ServiceBinding status is updated to `Malformed`.

### Update

When a ServiceBinding is updated, Primaza Application Agent will update the workload resources with the secret details.
If the secret is updated the projection in the workloads will be updated accordingly.
