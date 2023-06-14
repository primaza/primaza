Feature: Register a kubernetes cluster as Primaza Worker Cluster

    Background:
        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate

    Scenario: Primaza Cluster can contact Worker cluster, authentication is successful
        Given On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        When On Primaza Cluster "main", Resource is created
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
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"

    Scenario: Primaza Cluster can contact Worker cluster, but ClusterContext secret is missing
        When  On Primaza Cluster "main", Resource is created
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
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "False"

    Scenario: ServiceCatalog is created
        Given On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        When On Primaza Cluster "main", Resource is created
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
        Then On Primaza Cluster "main", ServiceCatalog "dev" exists

    Scenario: Events on secrets trigger ClusterEnvironment reconciliation
        Given On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Primaza Cluster "main", Resource is created
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
