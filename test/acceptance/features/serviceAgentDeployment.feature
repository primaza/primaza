@disabled
Feature: Publish Service Agent to worker cluster

    Scenario: On Cluster Environment creation, Primaza Service Agent is successfully deployed to services namespace

        Given Primaza Cluster "primaza-main" is running
        And Worker Cluster "primaza-worker" for "primaza-main" is running
        And Clusters "primaza-main" and "primaza-worker" can communicate
        And On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "primaza-worker", service namespace "services" exists
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
            serviceNamespaces:
            - "services"
        """
        Then On Worker Cluster "primaza-worker", Primaza Service Agent is deployed into namespace "services"

    Scenario: On Cluster Environment update, Primaza Service Agent is successfully removed from service namespace

        Given Primaza Cluster "primaza-main" is running
        And   Worker Cluster "primaza-worker" for "primaza-main" is running
        And   Clusters "primaza-main" and "primaza-worker" can communicate
        And   On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        And   On Worker Cluster "primaza-worker", service namespace "services" exists
        And   On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces:
            - services
        """
        And On Worker Cluster "primaza-worker", Primaza Service Agent is deployed into namespace "services"
        When On Primaza Cluster "primaza-main", Resource is updated
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces: []
        """
        Then On Worker Cluster "primaza-worker", Primaza Service Agent is not deployed into namespace "services"

    Scenario: On Cluster Environment update, Primaza Service Agent is successfully published into service namespace

        Given Primaza Cluster "primaza-main" is running
        And Worker Cluster "primaza-worker" for "primaza-main" is running
        And Clusters "primaza-main" and "primaza-worker" can communicate
        And On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "primaza-worker", service namespace "services" exists
        And On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces: []
        """
        When On Primaza Cluster "primaza-main", Resource is updated
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces:
            - services
        """
        Then On Worker Cluster "primaza-worker", Primaza Service Agent is deployed into namespace "services"
