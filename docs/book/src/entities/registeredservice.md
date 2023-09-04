# RegisteredService

A RegisteredService represents an instance of a running service.

This resource can be created by both a human operator or a piece of software (agent, bot, script).
This flexibility allows for the service provisioning and service discovery operations to be decoupled from representation of that service in a Service Catalog.

A RegisteredService can be claimed to be used by a specific application, and once claimed, the RegisteredService can't be claimed by other application.
If the application doesn't require the service any longer, it can remove the claim and the registered service is put back in the pool of available services.

## Specification

The definition of RegisteredServices can be obtained directly from [RegisteredService CRD](https://github.com/primaza/primaza/blob/main/config/crd/bases/primaza.io_registeredservices.yaml).

The RegisteredService's specification contains the following **required** properties:

- `serviceClassIdentity`: A set of key/value pairs that identify the service class.
Examples of service class identity keys include type of service, and provider of service.
This property is required.
- `serviceEndpointDefinition`: A set of key/value pairs that provides two pieces of information: the service connectivity and the service authentication.
Values can be either a string or a reference to a secret field.
This property is required.

A RegisteredService also has the following **optional** properties, which gives the user more control over the resource:

- `constraints`: Restrictions of the registered service.
  A constraint could be a specific environment where the service can't be claimed from.
  This property is optional, when it's absent, it means there isn't constraints.
  For more details, look at the [Constraints](#constraints) section.
- `healthcheck`: A mechanism to be able to verify the service is online and ready to use.
  One way this can be accomplished is by providing an image containing a client that can be run to test connectivity and authentication.
  This property is optional, when it's absent, it means the service will be considered available as soon as it's registered.
- `sla`: Provides multiple levels of resiliency, scalability, fault tolerance and security.
  This allows claims to consider the robustness of service.
  This property is optional, when it's absent, it means that there is no distinctions between services given the SLA.

### Constraints

The `constraints` section of a RegisteredService defines a list of environments.
The environment list can contain explicit environments that are to include the service, or it can also contain a list of environments that are to exclude the service.

The way to depict exclusion is by adding the `!` prefix to the environment, i.e `!prod`.
Please note that exclusions have precedence over inclusions.

For example, if the list contains `!prod` but also includes `dev`, then `dev` is considered to be in the `!prod` set of environments and therefore redundant.
If there is a third environment stage, then `!prod` would include both `stage` and `dev` even if they're not defined in the list explicitly.

## Metadata

A Primaza's discovered RegisteredService has the following annotations:

* `primaza.io/cluster-environment`: the name of the ClusterEnvironemnt it's part of
* `primaza.io/service-apiversion`: the APIVersion of the resource represented by the RegisteredService
* `primaza.io/service-kind`: the kind of the resource represented by the RegisteredService
* `primaza.io/service-name`: the name of the resource represented by the RegisteredService
* `primaza.io/service-namespace`: the namespace of the resource represented by the RegisteredService
* `primaza.io/service-uid`: the UID of the resource represented by the RegisteredService

## Status

The state of a RegisteredService could be one of the following:
- Registered
- Available
- Claimed
- Unreachable
- Undetermined

> As of now, health-check are still not implemented.
> The only states that are used are `Available` and `Claimed`.

The RegisteredService state is `Registered` right after creation.
Usually, this will change immediately after the controller gets notified of existence of new resource, unless controller fails even before it tries to run health-check.
If the health check passes or isn't defined, the state will change to `Available`.

Once a RegisteredService is claimed, its state moves to `Claimed`.
On the other hand, if the health-check comes back as a failure, the state would change to `Unreachable`.

Sometimes, the health check could fail before it can determine whether the service is healthy or not.
In those cases, the state of the RegisteredService will go to `Undetermined` indicating that there is an issue with the health-check and not with the service itself.

There are some situations when the health-check will fail while the RegisteredService is in `Claimed` state.
In those cases the RegisteredService will move to `Unreachable`.
If, at a later time, the health-check passes then the controller will check if there is still a claim matching the RegisteredService and move the state back to `Claimed`.
However, if there isn't claim matching the RegisteredService the state will move to `Available`.

## Use Cases

### Creation

When a RegisteredService is created, Primaza will first check if the health check can be run and will modify the state accordingly.
The ServiceCatalog will be updated to include the new RegisteredService only if the state can be moved to `Available`.

### Deletion

When a RegisteredService resource is deleted, the ServiceCatalog entry for the service should be deleted.
Also, if a RegisteredService is `Claimed`, the ServiceClaim resource state will be changed to `Pending` by the ServiceClaim controller since the matched RegisteredService doesn't exist any longer.
Additionally, when a RegisteredService resource state changes to `Claimed` the corresponding entry in the ServiceCatalog resource is removed.

### Update

When a RegisteredService is updated, a few things can happen:
- Existing ServiceClaim resources may not be able to match the registered service any longer.
In those case the ServiceClaim controller will try to find another match and, if unsuccessful, the ServiceClaim resource state will change to `Pending`.
- Existing service endpoint definition has changed and should trigger a binding secret generation.
