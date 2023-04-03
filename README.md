# Primaza

## Multi Cluster Service Consumption Framework

### The Problem
- Modern application stacks are increasingly complex
- Managing connections to all your supporting Services becomes more and more challenging… and that’s before you introduce additional clouds and hybrid environments
- No platform enables easy consumption of hybrid cloud services
- Needed to provide Scalability, Security and Portability
- Previous solutions (OSBAPI, Service Binding, etc.) gained moderate traction
  - Replace one complexity with another
  - Heavyweight, onerous, exacting implementations
- Customers are left to implement ad-hoc solutions
- Manual steps to provision and consume cloud service (not scalable)
- Custom IaC to provide the scale required (boilerplate imperative code)
- Too much time spent on non-value added activities

### The Solution

- Separated logical environments that can span across different kubernetes clusters
- An automatically collated Catalog that appropriately exposes available Services to Applications
- Leverages industry best practices to securely deliver service endpoint data to applications
- A platform that enables hybrid cloud services consumption in a portable, secured and scalable way

### Benefits to Service Providers

- No action required!
- Continue to focus on delivering their cloud service with well-defined APIs
- Continue to work with Service Kubernetes APIs and Controllers

### Benefits to Administrators

- Automatic addition of services to the catalog (via Service Class) - manual catalog management still possible (via Registered Service)
- Service Classes are easy to create, and can be built by the service provider, administrators, or third-parties
- Popular Service Classes delivered by Primaza out of the box

### Benefits to Developer
- Service Catalog makes it easy to discover new services and see what is available in the environment
- Developers can automatically and easily claim services from the catalog
- Claims can be requested for multiple environments (CI/CD capable)
- Developers can discover/learn via provided sandbox

## Implementation

[Primaza's architecture](./docs/architecture/agents.md) is composed by the following elements:
- Primaza control plane: manages environments, services and claims
- Application agents: binds applications to services
- Service agents: discover services


Primaza defines the following entities and controllers to provide the above described features.

Entities:
* [Cluster Environment](./docs/entities/clusterenvironment.md): represents an development environment on a kubernetes Cluster.
* [Registered Service](./docs/entities/registeredservice.md): represents running instance of a software service.
* [Service Binding](./docs/entities/servicebinding.md): projects secrets referenced by ServiceBinding resources to application compute resources.
* [Service Claim](./docs/entities/serviceclaim.md): represents a claim for Registered Service.
* [Service Catalog](./docs/entities/servicecatalog.md): represents group of Registered Services.

Controllers:
* Discovery Controller
* Claiming Controller
* Binding Controller

