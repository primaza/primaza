Feature: Delete ServiceCatalog on ClusterEnvironment deletion

    Background: Multi-cluster environment initializaiton
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Primaza Cluster "main", Resource is created
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
        And On Primaza Cluster "main", ServiceCatalog "dev" exists
    
    Scenario: ServiceCatalog is deleted
        When On Primaza Cluster "main", ClusterEnvironment "worker" is deleted
        Then On Primaza Cluster "main", ServiceCatalog "dev" does not exist

    Scenario: ServiceCatalog is still required
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker-2
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
        """
        When On Primaza Cluster "main", ClusterEnvironment "worker" is deleted
        Then On Primaza Cluster "main", ServiceCatalog "dev" exists
        