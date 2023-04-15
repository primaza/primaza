Feature: Publish Service Agent to worker cluster

    Scenario: On Cluster Environment creation, Primaza Service Agent is successfully deployed to services namespace

        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
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
            serviceNamespaces:
            - "services"
        """
        Then On Worker Cluster "worker", Primaza Service Agent exists into namespace "services"

    Scenario: On Cluster Environment update, Primaza Service Agent is successfully removed from service namespace

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
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
            serviceNamespaces:
            - services
        """
        And On Worker Cluster "worker", Primaza Service Agent exists into namespace "services"
        When On Primaza Cluster "main", Resource is updated
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces: []
        """
        Then On Worker Cluster "worker", Primaza Service Agent does not exist into namespace "services"

    Scenario: On Cluster Environment update, Primaza Service Agent is successfully published into service namespace

        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
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
            serviceNamespaces: []
        """
        When On Primaza Cluster "main", Resource is updated
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces:
            - services
        """
        Then On Worker Cluster "worker", Primaza Service Agent exists into namespace "services"

    Scenario: On Cluster Environment deletion, Primaza Service Agent is successfully removed from service namespace

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
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
            serviceNamespaces:
            - services
        """
        And On Worker Cluster "worker", Primaza Service Agent exists into namespace "services"
        When On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        """
        Then On Worker Cluster "worker", Primaza Service Agent does not exist into namespace "services"
