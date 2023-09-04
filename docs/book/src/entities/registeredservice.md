# RegisteredService

A RegisteredService represents an instance of a running service.

This resource can be created by both a human operator or a piece of software (agent, bot, script).
This flexibility allows for the service provisioning and service discovery operations to be decoupled from representation of that service in a Service Catalog.

A RegisteredService can be claimed to be used by a specific application, and once claimed, the RegisteredService cannot be claimed by other application.
If the application does not require the service any longer, it can remove the claim and the registered service is put back in the pool of available services.

## Specification

The definition of RegisteredServices can be obtained directly from our [RegisteredService CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_registeredservices.yaml).

The RegisteredService's specification contains the following **required** properties:

- `serviceClassIdentity`: A set of key/value pairs that identify the service class.
Examples of service class identity keys include type of service, and provider of service.
This property is required.
- `serviceEndpointDefinition`: A set of key/value pairs that provides two pieces of information: the service connectivity and the service authentication.
Values can be either a string or a reference to a secret field.
This property is required.

A RegisteredService also has the following **optional** properties, which gives the user more control over the resource:

- `constraints`: Restrictions of the registered service.
  A constraint could be a specific environment where the service cannot be claimed from.
  This property is optional, when it is absent, it means there is not constraints.
  For more details, look at the [Constraints](#constraints) section.
- `health check`: A mechanism to be able to verify the service is online and ready to use.
  One way this can be accomplished is by providing an image containing a client that can be run to test connectivity and authentication.
  This property is optional, when it is absent, it means the service will be considered available as soon as it is registered.
  For more information, see the section describing [health checks](#health-checks).
- `sla`: Provides multiple levels of resiliency, scalability, fault tolerance and security.
  This allows claims to take into account the robustness of service.
  This property is optional, when it is absent, it means that there is no distinctions between services given the SLA.

### Constraints

The `constraints` section of a RegisteredService defines a list of environments.
The environment list can contain explicit environments that are to include the service, or it can also contain a list of environments that are to exclude the service.

The way to depict exclusion is by adding the `!` prefix to the environment, i.e `!prod`.
Please note that exclusions have precedence over inclusions.

For example, if the list contains `!prod` but also includes `dev`, then `dev` is considered to be in the `!prod` set of environments and therefore redundant.
If there is a third environment stage, then `!prod` would include both `stage` and `dev` even if they are not defined in the list explicitly.

## Metadata

A Primaza's discovered RegisteredService has the following annotations:

* `primaza.io/cluster-environment`: the name of the ClusterEnvironemnt it's part of
* `primaza.io/service-apiversion`: the APIVersion of the resource represented by the RegisteredService
* `primaza.io/service-kind`: the kind of the resource represented by the RegisteredService
* `primaza.io/service-name`: the name of the resource represented by the RegisteredService
* `primaza.io/service-namespace`: the namespace of the resource represented by the RegisteredService
* `primaza.io/service-uid`: the UID of the resource represented by the RegisteredService

### Health checks

A registered service can specify a health check for the underlying service.
This check is used by Primaza to periodically check if the services is still actively available or not.

The health check contains the following fields underneath the `container` subfield:
- `image`: the container image to use.
- `command`: the command to be run in the provided image.
- `minutes`: the number of minutes to pass between checks.
  Defaults to 5 minutes, but can be set as low as 1.

An example health check that does nothing successfully:
```yaml
apiVersion: primaza.io/v1alpha1
kind: RegisteredService
metadata:
  name: service-1
spec:
  health check:
    container:
      image: "alpine:latest"
      command:
      - /bin/sh
      - -c
      - exit 0
  serviceClassIdentity:
    - name: type
      value: psqlserver
    - name: provider
      value: aws
  serviceEndpointDefinition:
    - name: port
      value: "5432"
```

Health checks are run inside [CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) with most privileges stripped.

## Status

The state of a RegisteredService could be one of the following:
- Available
- Claimed
- Unknown
- Unreachable

If a health check is not defined for the service, the state will transition to `Available`.
Otherwise, the state will be set to `Unknown` while the health check is run for the first time.
Once the health check completes successfully, the state will transition to `Available`.
However, if the health check indicates a failure, the state will change to `Unreachable`.

In some cases, Primaza cannot actively determine if the health check has succeeded or failed.
For instance, this can occur if the health check references a non-existent container image, but other cases may also trigger this behavior.
In those cases, the state of the RegisteredService will transition to `Unknown`, indicating that there may be a problem with the health check and that further investigation may be needed.

Once a RegisteredService is claimed with a [ServiceClaim][ServiceClaim], its state will transition to `Claimed`.
If the [ServiceClaim][ServiceClaim] is deleted, then the RegisteredService will transition from `Claimed` back to `Available`, making it available for re-use.

If the health check fails while the RegisteredService is in `Claimed` state, the RegisteredService will transition to `Unreachable`.
If, at a later time, the health check passes, then the controller will check if there is still a claim matching the RegisteredService and transition to `Claimed`.
Otherwise, the RegisteredService the state will transition to `Available`.

## Use Cases

### Creation

When a RegisteredService is created, Primaza will first check if the health check can be run and will modify the state accordingly.
The [ServiceCatalog][ServiceCatalog] will be updated to include the new RegisteredService only if the state can be moved to `Available`.

### Deletion

When a RegisteredService resource is deleted, the [ServiceCatalog][ServiceCatalog] entry for the service should be deleted.
Also, if a RegisteredService is `Claimed`, the [ServiceClaim] resource state will be changed to `Pending` by the [ServiceClaim][ServiceClaim] controller since the matched RegisteredService does not exist any longer.
Additionally, when a RegisteredService resource state changes to `Claimed` the corresponding entry in the [ServiceCatalog][ServiceCatalog] resource is removed.

### Update

When a RegisteredService is updated, a few things can happen:
- Existing [ServiceClaim][ServiceClaim] resources may not be able to match the registered service any longer.
In those case the [ServiceClaim][ServiceClaim] controller will try to find another match and, if unsuccessful, the [ServiceClaim][ServiceClaim] resource state will change to `Pending`.
- Existing service endpoint definition has changed and should trigger a binding secret generation.

[ServiceClaim]: ./serviceclaim.md
[ServiceCatalog]: ./servicecatalog.md
