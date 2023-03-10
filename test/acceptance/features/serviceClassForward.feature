Feature: Forward Service Class into Service namespaces

    Background:
        Given Primaza Cluster "primaza-main" is running
        And Worker Cluster "primaza-worker" for "primaza-main" is running
        And Clusters "primaza-main" and "primaza-worker" can communicate
        And On Primaza Cluster "primaza-main", Worker "primaza-worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "primaza-worker", service namespace "services" exists

    Scenario: On Service Class creation, Primaza control plane forwards it into all matching services namespace
        Given On Primaza Cluster "primaza-main", Resource is created
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
        And  On Worker Cluster "primaza-worker", Primaza Service Agent is deployed into namespace "services"
        When On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClass
        metadata:
            name: demo-service-sc
            namespace: primaza-system
        spec:
            constraints:
                environments:
                - dev
            resource:
                apiVersion: stable.example.com/v1
                kind: Backend
                serviceEndpointDefinitionMapping:
                    - name: host
                      jsonPath: .spec.host
            serviceClassIdentity:
                - name: type
                  value: backend
                - name: provider
                  value: stable.example.com
                - name: version
                  value: v1
        """
        Then On Worker Cluster "primaza-worker", Resource "ServiceClass" with name "demo-service-sc" exists in namespace "services"

    Scenario: Service Classes are pushed to new Cluster Environments' service namespaces
        Given   On Primaza Cluster "primaza-main", Resource is created
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
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Online"
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "Online" has Status "True"
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClass
        metadata:
            name: demo-serviceclass
            namespace: primaza-system
        spec:
            constraints: {}
            resource:
                apiVersion: stable.example.com/v1
                kind: Backend
                serviceEndpointDefinitionMapping:
                - name: host
                  jsonPath: .spec.host
            serviceClassIdentity:
            - name: type
              value: backend
            - name: provider
              value: stable.example.com
            - name: version
              value: v1
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
        Then On Worker Cluster "primaza-worker", Service Class "demo-serviceclass" exists in "services"

    Scenario: Service Classes are pushed to service namespaces of new Cluster Environments
        Given On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClass
        metadata:
            name: demo-serviceclass
            namespace: primaza-system
        spec:
            constraints: {}
            resource:
                apiVersion: stable.example.com/v1
                kind: Backend
                serviceEndpointDefinitionMapping:
                - name: host
                  jsonPath: .spec.host
            serviceClassIdentity:
            - name: type
              value: backend
            - name: provider
              value: stable.example.com
            - name: version
              value: v1
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
            serviceNamespaces:
            - services
        """
        Then On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" state will eventually move to "Online"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "primaza-main", ClusterEnvironment "primaza-worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And  On Worker Cluster "primaza-worker", Service Class "demo-serviceclass" exists in "services"
