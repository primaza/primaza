Feature: Register a kubernetes cluster as Primaza Worker Cluster

    Scenario: Cluster Environment status is Partial if Application namespaces permissions are missing

        Given Primaza Cluster "primaza-main" is running
        And   Worker Cluster "primaza-worker" for "primaza-main" is running
        And   Clusters "primaza-main" and "primaza-worker" can communicate
        And   On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        And   On Worker Cluster "primaza-worker", application namespace "applications" exists
        And   On Worker Cluster "primaza-worker", Resource is deleted
        """
        apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        metadata:
            name: primaza-rolebinding
            namespace: applications
        """
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
            applicationNamespaces:
            - applications
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Partial"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "True"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"

    Scenario: Cluster Environment status is Online if Application namespaces permissions are present

        Given Primaza Cluster "primaza-main" is running
        And   Worker Cluster "primaza-worker" for "primaza-main" is running
        And   Clusters "primaza-main" and "primaza-worker" can communicate
        And   On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        And   On Worker Cluster "primaza-worker", application namespace "applications" exists
        When  On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: primaza-worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            applicationNamespaces:
            - applications
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Online"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
