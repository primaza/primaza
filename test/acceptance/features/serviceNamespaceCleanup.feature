Feature: Cleanup service namespace

    Background:
        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
        And   On Worker Cluster "worker", Primaza Service Agent is deployed into namespace "services"
        And   Resource "backend_crd.yaml" is installed on worker cluster "worker" in namespace "services"
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
                serviceNamespaces:
                - services
            """
        And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And On Primaza Cluster "main", Resource is created
            """
            apiVersion: primaza.io/v1alpha1
            kind: ServiceClass
            metadata:
                name: $scenario_id-serviceclass
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
        And  On Worker Cluster "worker", Service Class "$scenario_id-serviceclass" exists in "services"

    Scenario: Service Class is removed on Service Namespace deletion
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
            """
        Then On Worker Cluster "worker", Service Class "$scenario_id-serviceclass" does not exist in "services"

    Scenario: Service Class is removed on Cluster Environment deletion
        When On Primaza Cluster "main", Resource is deleted
            """
            apiVersion: primaza.io/v1alpha1
            kind: ClusterEnvironment
            metadata:
                name: worker
                namespace: primaza-system
            """
        Then On Worker Cluster "worker", Service Class "$scenario_id-serviceclass" does not exist in "services"
