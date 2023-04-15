Feature: Register a kubernetes cluster as Primaza Worker Cluster

    Scenario: Primaza Cluster can contact Worker cluster, authentication is successful
        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
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
        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
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

    Scenario: Cluster Environment is created outside of Primaza's namespace
        Given Primaza Cluster "main" is running
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: v1
        kind: Namespace
        metadata:
            name: out-of-scope-namespace
        ---
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: out-of-scope-namespace
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" in namespace "out-of-scope-namespace" state remains not present

    Scenario: ServiceCatalog is created
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
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

