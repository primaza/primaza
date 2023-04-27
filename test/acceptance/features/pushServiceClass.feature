Feature: Create or Update Service Class

    Background:
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
            serviceNamespaces:
            - "services"
        """
        And On Worker Cluster "worker", Primaza Service Agent exists into namespace "services"
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
        And On Worker Cluster "worker", Resource "ServiceClass" with name "demo-service-sc" exists in namespace "services"
        And jsonpath ".spec.serviceClassIdentity" on "serviceclasses.primaza.io/demo-service-sc:services" in cluster worker is "[{"name":"type", "value":"backend"}, {"name":"provider", "value":"stable.example.com"}, {"name":"version", "value":"v1"}]"


    Scenario: On Service Class update, Primaza control plane forwards it into all matching services namespace
        When On Primaza Cluster "main", Resource is updated
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
                  value: backend-update
                - name: provider
                  value: stable.example.com
                - name: version
                  value: v1
        """
        Then jsonpath ".spec.serviceClassIdentity" on "serviceclasses.primaza.io/demo-service-sc:services" in cluster worker is "[{"name":"type", "value":"backend-update"}, {"name":"provider", "value":"stable.example.com"}, {"name":"version","value":"v1"}]" 
