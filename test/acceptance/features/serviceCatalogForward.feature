Feature: Reconcile Service Catalog in Application namespaces
    Scenario: Service Catalog is pushed to new Cluster Environments' application namespaces
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
        """
        And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: primaza-rsdb
          namespace: primaza-system
        spec:
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: host
              value: mydavphost.io
            - name: port
              value: "5432"
            - name: user
              value: davp
            - name: password
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
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
        Then On Worker Cluster "worker", ServiceCatalog "dev" exists in "applications"

    Scenario: Service Catalog is pushed to application namespaces of new Cluster Environments
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: primaza-rsdb
          namespace: primaza-system
        spec:
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: host
              value: mydavphost.io
            - name: port
              value: "5432"
            - name: user
              value: davp
            - name: password
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
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
            applicationNamespaces:
            - applications
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And On Worker Cluster "worker", ServiceCatalog "dev" exists in "applications"