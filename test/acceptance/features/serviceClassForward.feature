Feature: Forward Service Class into Service namespaces

    Scenario: On Service Class creation, Primaza control plane forwards it into all matching services namespace

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
