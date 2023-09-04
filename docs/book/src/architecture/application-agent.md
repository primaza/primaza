## Application Agent

Application Agents are installed into ClusterEnvironment's Application Namespaces.
Target namespaces need to be already configured to allow agents to run.

Application Agents need to access resources in the namespace they're published into.
More specifically, an Application Agent requires the following resources to exists into the namespace:

* A Role granting
    * full access to `leases.coordination.k8s.io`
    * read access to `servicebindings.primaza.io`
    * read access and update rights for `deployments.apps`
    * create right for `events`
* A Service Account for the agent
* A RoleBinding that binds the ServiceAccount to the Role
* A Secret with the kubeconfig to communicate back with Primaza's Control Plane

To prepare Application Namespaces you can use [primazactl](https://github.com/primaza/primazactl).

When a ServiceBinding is created in an Application Namespace, the Application Agent looks for resources mentioned in its specification.

Primaza Application Agent runs a dynamic informer for `Application` resources mentioned in the ServiceBinding's specification.
The informer monitors changes to the `Application` matching the ServiceBinding specifications and updates the ServiceBinding's status accordingly.

* If the `Application` Resource mentioned in ServiceBinding specification is updated or created, the secret referenced by ServiceBinding resource will be projected into all the matching applications.
* If the `Application` Resource is deleted and no matching workloads are found in the namespace, then the ServiceBinding status condition `Reason` is updated to `NoMatchingWorkloads`.


### Binding a Service

When a ServiceBinding is created (or updated) into an Application namespace, the Application Agent gets the data from the secrets and project them into applications specified in the ServiceBinding instance.

Currently the secret data is being projected as volume mounts.
`SERVICE_BINDING_ROOT` points to the environment variable in the container which is used as the volume mount path.
In the absence of this environment variable, `/bindings` is used as the volume mount path.

Please refer to https://github.com/servicebinding/spec#reconciler-implementation for more information.

### Claiming a Service

#### Claiming from Primaza's Control Plane Namespace

When a ServiceClaim is created in Primaza's Control Plane's Namespace, Primaza builds the ServiceBinding and Service Endpoint Definition Secrets and pushes them resources to all the application namespaces of matching ClusterEnvironments.
The ServiceClaim controller in Primaza Control Plane watches RegisteredService resources.
Any change made to `ServiceEndpointDefinition` values in RegisteredService are propagated to the secret and ServiceBinding resource in application namespace by Primaza.

#### Claiming from Application Namespace

When a ServiceClaim is created in Primaza's Control Plane's Namespace, the Application Agent forwards this ServiceClaim to Primaza's Control Plane.
The Control Plane, in turn, builds the ServiceBinding and Service Endpoint Definition Secrets and pushes them back to the Application Namespace.
