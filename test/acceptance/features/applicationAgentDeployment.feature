Feature: Publish Application Agent to worker cluster

    Scenario: On Cluster Environment creation, Primaza Application Agent is successfully deployed to applications namespace
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
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
            applicationNamespaces:
            - "applications"
        """
        Then On Worker Cluster "worker", Primaza Application Agent exists into namespace "applications"

    Scenario: On Cluster Environment update, Primaza Application Agent is successfully removed from application namespace

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
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
            applicationNamespaces:
            - applications
        """
        And On Worker Cluster "worker", Primaza Application Agent exists into namespace "applications"
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
            applicationNamespaces: []
        """
        Then On Worker Cluster "worker", Primaza Application Agent does not exist into namespace "applications"

    Scenario: On Cluster Environment update, Primaza Application Agent is successfully published into application namespace

        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
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
            applicationNamespaces: []
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
            applicationNamespaces:
            - applications
        """
        Then On Worker Cluster "worker", Primaza Application Agent exists into namespace "applications"

    Scenario: On Cluster Environment deletion, Primaza Application Agent is successfully removed from application namespace

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
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
            applicationNamespaces:
            - applications
        """
        And On Worker Cluster "worker", Primaza Application Agent exists into namespace "applications"
        When On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        """
        Then On Worker Cluster "worker", Primaza Application Agent does not exist into namespace "applications"


