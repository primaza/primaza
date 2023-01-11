Feature: Register a kubernetes cluster as Primaza Worker Cluster

    Scenario: Primaza Cluster can contact Worker cluster, authentication is successful
        Given Primaza Cluster "primaza-main" is running
        And   Worker Cluster "primaza-worker" for "primaza-main" is running
        And   Clusters "primaza-main" and "primaza-worker" can communicate
        And   On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        When On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Online"
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" last status condition has Type "Online"

    Scenario: Primaza Cluster can contact Worker cluster, but ClusterContext secret is missing
        Given Primaza Cluster "primaza-main" is running
        And   Worker Cluster "primaza-worker" for "primaza-main" is running
        And   Clusters "primaza-main" and "primaza-worker" can communicate
        When On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Offline"
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" last status condition has Type "Offline"

    Scenario: Primaza Cluster can contact Worker cluster, but has invalid credentials
        Given Primaza Cluster "primaza-main" is running
        And   Worker Cluster "primaza-worker" for "primaza-main" is running
        And   Clusters "primaza-main" and "primaza-worker" can communicate
        And   On Primaza Cluster "primaza-main", an invalid Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        When On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Offline"
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" last status condition has Type "Offline"

    Scenario: Cluster Environment is created outside of Primaza's namespace
        Given Primaza Cluster "primaza-main" is running
        When On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: v1
        kind: Namespace
        metadata:
            name: out-of-scope-namespace
        ---
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: out-of-scope-namespace
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" in namespace "out-of-scope-namespace" state remains not present
