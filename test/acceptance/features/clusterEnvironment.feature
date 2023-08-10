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

    Scenario: Primaza Worker cluster has excess permission than required
        Given On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
        And On Primaza Cluster "worker", Resource is created
        """
        apiVersion: rbac.authorization.k8s.io/v1
        kind: Role
        metadata:
            namespace: applications
            name: pod-reader
        rules:
        - apiGroups: [""]
          resources: ["pods"]
          verbs: ["get", "watch", "list"]
        """
        And On Primaza Cluster "worker", Resource is created
        """
        apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        metadata:
            name: pod-reader
            namespace: applications
        roleRef:
            apiGroup: rbac.authorization.k8s.io
            kind: Role
            name: pod-reader
        subjects:
        - kind: ServiceAccount
          name: primaza-primaza-system-worker
          namespace: kube-system
        """
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
            applicationNamespaces:
            - applications
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ExcessPermissionsInApplicationNamespaces" has Status "True"
