# Claiming

Once you have a RegisteredService, you can claim it for one or more workloads.
The following examples uses the service from the [Manual Registration tutorial](../discovery/manual-registration.md).

A ServiceClaim can select the workload to use filtering by [name](./application-name.md) or [labels](./labels.md).

Finally, you can claim from the Control Plane or from an Application Namespace.

| Workflow              | Single Namespace | All Namespaces | By Label | By Name |
|-----------------------|------------------|----------------|----------|---------|
| Control Plane         | ✅               | ✅             | ✅       | ✅      |
| Application Namespace | ✅               | ❌             | ✅       | ✅      |


## Control Plane Workflow

If you create the ServiceClaim in the Control Plane you can define whether it applies to all the Application Namespaces in the Environment or it's just for one specific namespace.
This filter is implemented by the mean of the Claim's property `applicationClusterContext`.

For examples refer to [Filtering by Name](./filtering-by-name.md) or [Filtering by Labels](./filtering-by-labels.md) tutorials for Environment wide ServiceClaim examples, and to [Application Namespace Scoped](./application-namespace-scoped.md) tutorial.

## Application Namespace Workflow

When a Service Claim is created in an Application Namespace, Primaza pushes the ServiceBinding and Secret just to this namespace.

In this case, indeed, the Application Agent tampers and send the ServiceClaim to the Control Plane.
The Control Plane, in turn, pushes the ServiceBinding and Secret only to the ApplicationNamespace in which the ServiceClaim was created.
