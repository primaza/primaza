# Architecture

Primaza's architecture is composed by the following elements:
- Primaza's Control Plane: manages environments, services and claims
- Application agents: binds applications to services
- Service agents: discover services

To better describe the Primaza architecture we can introduce the following three concepts:
- Primaza's Control Plane Namespace: a namespace where Primaza is installed
- Application Namespace: a namespace configured to host the Primaza's Application Agent
- Service Namespace: a namespace configured to host the Primaza's Service Agent

For how Primaza is designed, these three concepts may apply at the same time to a single namespace.
In other words, we can install Primaza, Primaza's Application Agent, and Primaza's Service Agent in the same kubernetes namespace.

In the following picture you find a simplified diagrams of the agents-based architecture.

![image](../imgs/architecture-agents-simplified.png)

To allow agents to perform operations in the namespace, they need an Service Account with the right permissions.

[primazactl](https://github.com/primaza/primazactl) is an in-development companion tool to help administrators configuring clusters and namespaces.

In the following picture, you find a detailed representation of all the resources created in a Primaza environment.

![image](../imgs/architecture-agents-detailed.png)
