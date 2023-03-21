Feature: Forward Service Class into Service namespaces

    Background:
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", service namespace "services" exists

    Scenario: On Service Class creation, Primaza control plane forwards it into all matching services namespace
        Given On Primaza Cluster "main", Resource is created
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
        And  On Worker Cluster "worker", Primaza Service Agent is deployed into namespace "services"
        When On Primaza Cluster "main", Resource is created
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
        Then On Worker Cluster "worker", Resource "ServiceClass" with name "demo-service-sc" exists in namespace "services"

    Scenario: Service Classes are pushed to new Cluster Environments' service namespaces
        Given   On Primaza Cluster "main", Resource is created
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
        Then On Worker Cluster "worker", Service Class "demo-serviceclass" exists in "services"

    Scenario: Service Classes are pushed to service namespaces of new Cluster Environments
        Given On Primaza Cluster "main", Resource is created
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
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And  On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And  On Worker Cluster "worker", Service Class "demo-serviceclass" exists in "services"

    Scenario: On Service Class deletion, Primaza control plane deletes it from all matching services namespace
        Given On Primaza Cluster "main", Resource is created
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
        And On Worker Cluster "worker", Primaza Service Agent is deployed into namespace "services"
        And On Primaza Cluster "main", Resource is created
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
        And  On Worker Cluster "worker", Resource "ServiceClass" with name "demo-service-sc" exists in namespace "services"
        When On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClass
        metadata:
            name: demo-service-sc
            namespace: primaza-system
        """
        Then On Worker Cluster "worker", Resource "ServiceClass" with name "demo-service-sc" does not exist in namespace "services"
