# ServiceBinding

A Service Binding represents the secrets along with the Applications which are to bound with it. Service Bindings may explicitly request an Application by Name or by LabelSelector. LabelSelector can match more than one resource. Service Binding resource is being created by the `ServiceClaim` controller. Service Binding is being performed by [Primaza Application Agent](../architecture/agents.md) , and the agent should be deployed in the Application namespace where the workloads are to be bound. When a Service Binding is created (or updated) into an Application namespace, the Application agent has to get the data from the secrets and project them into applications matching the Application specification from Service Binding instance. Currently the secret data is being projected as volume mounts. `SERVICE_BINDING_ROOT` points to the environment variable in the container which is used as the volume mount path.  In the absence of this environment variable, `/bindings` is used as the volume mount path. Please Refer to https://github.com/servicebinding/spec#reconciler-implementation for more information.


## Specification

The definition of a Service Binding can be obtained directly from our [ServiceBinding CRD](../../config/crd/bases/primaza.io_servicebindings.yaml). The specification contains two required properties:

`ServiceEndpointDefinitionSecret`: ServiceEndpointDefinitionSecret is the name of the secret to project into the application. This property is required.
`Application`: 	Application resource to inject the binding info. It could be any process running within a container. A `ServiceBinding` **MAY** define the application reference by-name or by-[label selector][ls]. A name and selector are mutually exclusive.

## Status

The status of a Service Binding is also defined in our [ServiceBinding CRD](../../config/crd/bases/primaza.io_servicebindings.yaml). Service Binding status contains `state` and a `conditions` list.

The state of the service binding can be `Malformed` or `Ready`. The default value of service binding state is `Malformed`.

The `conditions` list of the service binding contains the following properties:
- `Type`: The service binding condition type is `Bound` or `NotBound`.
`Bound` means that the secret is projected into the application.
`NotBound` denotes that the secret is not projected into the application.
This can only occur if the secret is not found in the application namespace.
- `Message`: This contains the error logs for the service binding resources. This value will be an empty string if successful
- `Status`: Status of service binding can be `True` or `False`
- `Reason`: The reason has values defined as `NoMatchingWorkloads`, `ErrorFetchSecret`, `Successful` and `Binding Failure`

## Use Cases

### Creation

When a Service Binding is created, the secret referenced by the Service Binding itself will be projected into all the matching applications.
Matching applications are calculated as defined at in the section [Specification](#specification)

### Deletion

If the Service Binding is deleted the secret projection from the workloads will be removed.
In case the secret referenced in the Service Binding resource is deleted, the projection is removed from the workloads and the Service Binding status is updated to `Malformed`.

### Update

When a Service Binding is updated, Primaza Application Agent will update the workload resources with the secret details.
If the secret is updated the projection in the workloads will be updated accordingly.

