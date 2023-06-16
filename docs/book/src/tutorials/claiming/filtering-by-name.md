# By Application Name

When claiming a service, you can explicitly declare the target workload for the binding.
Primaza's Application Agent will project binding data just into that workload.

To obtain this result, you need to declare in the ServiceClaim how to identify the workload using the `application` property.

## Tutorial

{{#tutorial ../../../hack/tutorials/claiming/filtering-by-name.sh}}
