## Service Namespace Reconciliation

The Lifecycle of a Service Namespace includes the creation, update and deletion of Primaza's resources.

Namespaces are reconciled by the ClusterEnvironment controllers.
This controller runs in the Primaza's Control Plane.

Events on Service Namespaces are strictly related to ones on ClusterEnvironments.

* Service Namespace Creation: when a service namespace is added to the `serviceNamespaces` list of a ClusterEnvironment, the ClusterEnvironment controller checks the permissions in the target namespace and tries to push the Service Agent in it.
Finally, it will push in the namespace all the matching ServiceClasses for the given environment.

* Service Namespace Update: when an update event on the Service Namespace parent  ClusterEnvironment occurs, all service namespaces are reconciled as described in the `Service Namespace Creation` and `Service Namespace Deletion`.

* Service Namespace Deletion: when an Service Namespace is deleted from its parent ClusterEnvironment's `serviceNamespaces` list or when the ClusterEnvironment itself is deleted, the Service Agent is deleted from the target namespace.
As a result, this will trigger a deletion of all the Primaza's resources in the namespace.
This deletion is actually relying on kubernetes' ownership.
