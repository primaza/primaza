# Application Namespace Scoped

You can create a ServiceClaim that applies to a single Application Namespace in the Environment.
To obtain this result, you need to declare in the application namespace using the `applicationClusterContext` property.

This has no implication on workload filtering, so you can identify workloads in the Application Namespace either [by Name](./filtering-by-name.md) or [by Labels](./filtering-by-labels.md).

## Tutorial

{{#tutorial ../../../hack/tutorials/claiming/application-namespace-scoped.sh}}
