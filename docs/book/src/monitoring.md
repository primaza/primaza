# Monitoring

Primaza's resources are enriched with metadata that can be used to monitor the connections between workloads and services.

* Discovered [RegisteredServices](entities/registeredservice.md) takes track of the Service they represent in their `annotations`.
* [ServiceBindings](entities/servicebinding.md) take note of the RegisteredService they are related in their `annotations` and the workloads it tampered in the `status`.
* [ServiceClaims](entities/serviceclaim.md) stores info about the claimed RegisteredService in its `status`
