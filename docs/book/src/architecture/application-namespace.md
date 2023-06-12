## Application Namespace Reconciliation

The Lifecycle of a Application Namespace includes the creation, update and deletion of Primaza's resources.

Application Namespaces are reconciled by the ClusterEnvironment controllers.
This controller runs in the Primaza's Control Plane.

Events on Application Namespaces are strictly related to ones on ClusterEnvironments.

* Application Namespace Creation: when an application namespace is added to the `applicationNamespaces` list of a ClusterEnvironment, the ClusterEnvironment controller checks the permissions in the target namespace and tries to push the Application Agent in it.
Finally, it will push in the namespace the ServiceCatalog for the given environment and all the ServiceBinding generated from ServiceClaims declaring a `labelMatch` or that are targeting the namespace in the `applicationContext` property.

* Application Namespace Update: when an update event on the Application Namespace parent  ClusterEnvironment occurs, all application namespaces are reconciled as described in the `Application Namespace Creation` and `Application Namespace Deletion`.

* Application Namespace Deletion: when an Application Namespace is deleted from its parent ClusterEnvironment's `applicationNamespaces` list or when the ClusterEnvironment itself is deleted, the Application Agent is deleted from the target namespace.
As a result, this will trigger a deletion of all the Primaza's resources in the namespace.
This deletion is actually relying on kubernetes' ownership.
