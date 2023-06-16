# From Application Namespace

You can create a ServiceClaim directly from an Application Namespace.
When a Service Claim is created in an Application Namespace, Primaza pushes the ServiceBinding and Secret just to this namespace.

In this case, indeed, the Application Agent tampers and send the ServiceClaim to the Control Plane.
The Control Plane, in turn, pushes the ServiceBinding and Secret only to the ApplicationNamespace in which the ServiceClaim was created.

This has no implication on workload filtering, so you can identify workloads in the Application Namespace either [by Name](./filtering-by-name.md) or [by Labels](./filtering-by-labels.md).


## Tutorial

{{#tutorial ../../../hack/tutorials/claiming/from-application-namespace.sh}}
