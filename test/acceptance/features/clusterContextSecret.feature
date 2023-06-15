Feature: ClusterEnvironment is reconciled on ClusterContext secret events

    Background:
        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate

    Scenario: Create Event on ClusterContext secret triggers ClusterEnvironment reconciliation
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        And  On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "False"
        When On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"

    Scenario: Update event on ClusterContext secret triggers ClusterEnvironment reconciliation
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        And  On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And  On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        When On Primaza Cluster "main", "secret" named "primaza-kw" in "primaza-system" is patched
        """
        {
            "data": {
                "kubeconfig": ""
            }
        }
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "False"

    Scenario: Delete event on ClusterContext secret triggers ClusterEnvironment reconciliation
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And  On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        When On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: primaza-kw
            namespace: primaza-system
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "False"

