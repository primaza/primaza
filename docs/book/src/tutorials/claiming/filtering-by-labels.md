# By Application Name

When claiming a service, you can identify the target workloads for the binding by declaring the labels they should have.
Primaza's Application Agent will project binding data just into all matching workloads.

To obtain this result, you need to declare in the ServiceClaim how to identify the workload using the `application` property.

## Tutorial

{{#tutorial ../../../hack/tutorials/claiming/filtering-by-labels.sh}}
