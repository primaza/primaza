## Control Plane

The Primaza's Control Plane is the core of Primaza.

It contains all the logic for the management of the following resources:
* [Cluster Environment](./entities/clusterenvironment.md): represents an development environment on a kubernetes Cluster.
* [Registered Service](./entities/registeredservice.md): represents running instance of a software service.
* [Service Binding](./entities/servicebinding.md): projects secrets referenced by ServiceBinding resources to application compute resources.
* [Service Class](./entities/serviceclass.md): defines how a registered service can be automatically generated from a service
* [Service Claim](./entities/serviceclaim.md): represents a claim for Registered Service.
* [Service Catalog](./entities/servicecatalog.md): represents group of Registered Services.
