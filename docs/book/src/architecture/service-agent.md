## Service Agent

Service Agents are installed into ClusterEnvironment's Service Namespaces.
Target namespaces need to be already configured to allow agents to run properly.
Service Agents just need to access resources in the namespace they are published into.

More specifically, a Service agent requires the following resources to exists into the namespace:

* A Role granting
    * full access to `leases.coordination.k8s.io`
    * create right for `events`
* A Service Account for the agent
* A RoleBinding that binds the ServiceAccount to the Role
* A Secret with the kubeconfig to communicate back with Primaza's Control Plane

To easily prepare Service Namespaces you can use [primazactl](https://github.com/primaza/primazactl).

When a ServiceClass is created, the Service Agent looks for resources matching its specification.

A Role needs to be created which allows to retrieve, list and watch ServiceClass resources as Primaza's Service Agent runs a dynamic informer for each resource.

The informer monitors changes to resources matching the ServiceClass specifications and updates the RegisteredServices on Primaza control plane.

### Service Discovery

The Service Agent monitors all the resources specified in Service Classes existing in its namespace.
When a resource matching a Service Class is created, updated, or deleted, the Service Agent is notified and will create a RegisteredService in Primaza's Control Plane.
