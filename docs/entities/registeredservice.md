# RegisteredService

A RegisteredService represents an instance of a running service.
This resource can be created by both a human operator or a piece of software (agent, bot, script). This flexibility allows for the service provisioning and service discovery operations to be decoupled from representation of that service in a service catalog.
The service catalog can be consumed by Kubernetes applications that are unaware of the lifecycle of the service.
The creation of a registered service signals an update to the service catalog.
The service catalog can then be advertised to the application developers.
A RegisteredService can be claimed to be used by a specific application, and once claimed, the RegisteredService cannot be claimed by other application.
If the application does not require the service any longer, it can remove the claim and the registered service is put back in the pool of available services.

## Specification

The definition of a RegisteredService can be obtained directly from our [RegisteredService CRD](../../config/crd/bases/primaza.io_registeredservices.yaml). The specification contains two required properties:

- ServiceClassIdentity: A set of key/value pairs that identify the service class. Examples of service class identity keys include type of service, and provider of service.
This property is required.
- ServiceEndpointDefinition: A set of key/value pairs that provides two pieces of information: the service connectivity and the service authentication.
Values can be either a string or a reference to a secret field.
This property is required.

A RegisteredService also has three optional properties, which gives the user more control over the resource:

- Constraints: Restrictions of the registered service.
A constraint could be a specific environment where the service cannot be claimed from.
As an example, a constraint could describe that I don't want this service to be used on any environment except for production.
This property is optional, when it is absent, it means there is not constraints.
- HealthCheck: A mechanism to be able to verify the service is online and ready to use.
One way this can be accomplished is by providing an image containing a client that can be run to test connectivity and authentication. This property is optional, when it is absent, it means the service will be considered available as soon as it is registered.
- SLA: Provides multiple levels of resiliency, scalability, fault tolerance and security. This allows claims to take into account the robustness of service. This property is optional, when it is absent, it means that there is no distinctions between services given the SLA.


## Status

The Status of the Service it is also defined under the [RegisteredService CRD](../../config/crd/bases/primaza.io_registeredservices.yaml).
It contains only one required property to track state.
The state could be one of the following: registered, available, claimed, unreachable or undetermined.
The registered service state is "registered" right after creation.
In most cases this will change immediately after the controller gets notified of existence of new resource, unless controller fails even before it tries to run health check.
If the health check passes or is not defined, the state will change to "available".
Once a RegisteredService is claimed, its state moves to "claimed".
On the other hand, if the health check comes back as a failure, the state would change to "unreachable".
In some cases, the health check could fail before it can determine whether the service is healthy or not.
In those cases, the state of the registered service will go to "undetermined" indicating there is an issue with the health check and not with the service itself.

There are some situations when the health check will fail while the registered service is in "claimed" state.
In those cases the registered service will move to "unreachable".
If, at a later time, the health check passes then the controller will check if there is still a claim matching the registered service and move the state back to "claimed".
However, if there is not claim matching the registered service the state will move to "available"

## Use Cases

### Creation

When a RegisteredService is created, Primaza will first check if the health check can be run and will modify the state accordingly.
The ServiceCatalog will be updated to include the new RegisteredService only if the state can be moved to "available".

### Deletion

When a RegisteredService resource is deleted, the ServiceCatalog entry for the service should be deleted.
Also, if a RegisteredService is claimed, the ServiceClaim resource state will be changed to "Pending" by the ServiceClaim controller since the matched RegisteredService does not exist any longer.
Additionally, when a RegisteredService resource state changes to "claimed" the corresponding entry in the ServiceCatalog resource is removed.

### Update

When a RegisteredService is updated, a few things can happen:
- Existing ServiceClaim resources may not be able to match the registered service any longer.
In those case the service claim controller will try to find another match and, if unsuccessful, the ServiceClaim resource state will change to "pending".
- Existing service endpoint definition has changed and should trigger a binding secret generation. 
