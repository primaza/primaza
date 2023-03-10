Feature: Register a kubernetes cluster as Primaza Worker Cluster

    Scenario: Cluster Environment status is Partial if Service namespaces permissions are missing

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for "main" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And   On Worker Cluster "worker", service namespace "services" exists
        And   On Worker Cluster "worker", Resource is deleted
        """
        apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        metadata:
            name: primaza-rolebinding
            namespace: services
        """
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
            - services
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Partial"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"

    Scenario: Cluster Environment status is Online if Service namespaces permissions are present

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for "main" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And   On Worker Cluster "worker", service namespace "services" exists
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
            serviceNamespaces:
            - services
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
