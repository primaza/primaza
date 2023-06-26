# Namespace Reconciliation

There two types of Namespaces defined in Primaza, where the reconciliation happens.
Namespaces are reconciled by the ClusterEnvironment controllers which runs in the Primaza's Control Plane. 

* Application Namespace reconciliation: Creation, Update and Deletion of the resources in application namespace
* Service Namespace reconciliation: Creation, Update and Deletion of the resources in service namespace

During the Namespace Lifecycle, Primaza's permissions in the namespace are checked, permissions for talking with Primaza's Control Plane are set, and then namespace agent is deployed into the namespace.

Later, during Agent deletion, its permissions on Primaza's Control Plane are revoked.
