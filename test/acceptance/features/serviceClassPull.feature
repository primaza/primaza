Feature: Use ServicesClass resources to manage RegisteredService resources (Pull)

    Background:
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
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
            synchronizationStrategy: Pull
            serviceNamespaces:
            - services
        """
        And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
        And On Worker Cluster "worker", Primaza Service Agent exists into namespace "services"
        And Resource "backend_crd.yaml" is installed on worker cluster "worker" in namespace "services"
        And On Worker Cluster "worker", Resource is deleted
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: primaza-svc-kubeconfig
            namespace: services
        """

    Scenario: A Service Class creates Registered Services as specified
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id-1
                namespace: services
            spec:
                host: internal.db.stable.example.com
            """
        And On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id-2
                namespace: services
            spec:
                host: external.db.stable.example.com
            """
        When On Worker Cluster "worker", Resource is created
            """
            apiVersion: primaza.io/v1alpha1
            kind: ServiceClass
            metadata:
                name: $scenario_id-serviceclass
                namespace: services
            spec:
                constraints: {}
                resource:
                    apiVersion: stable.example.com/v1
                    kind: Backend
                    serviceEndpointDefinitionMappings:
                        resourceFields:
                        - name: host
                          jsonPath: .spec.host
                          secret: false
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        Then The resource registeredservices.primaza.io/$scenario_id-1:primaza-system is available in cluster "main"
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-1:primaza-system" in cluster main is "{"name":"host","value":"internal.db.stable.example.com"}"
        And The resource registeredservices.primaza.io/$scenario_id-2:primaza-system is available in cluster "main"
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "{"name":"host","value":"external.db.stable.example.com"}"
        And jsonpath ".metadata.annotations."primaza.io/service-apiversion"" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "stable.example.com/v1"
        And jsonpath ".metadata.annotations."primaza.io/service-kind"" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "Backend"
        And jsonpath ".metadata.annotations."primaza.io/service-name"" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "$scenario_id-2"
        And jsonpath ".metadata.annotations."primaza.io/service-namespace"" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "services"
        And jsonpath ".metadata.annotations."primaza.io/cluster-environment"" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "worker"

    Scenario: A Registered Service should not be created if the resource doesn't have the needed binding information
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id-1
                namespace: services
            spec:
                host_internal_db: internal.db.stable.example.com
            """
        When On Worker Cluster "worker", Resource is created
            """
            apiVersion: primaza.io/v1alpha1
            kind: ServiceClass
            metadata:
                name: $scenario_id-serviceclass
                namespace: services
            spec:
                constraints: {}
                resource:
                    apiVersion: stable.example.com/v1
                    kind: Backend
                    serviceEndpointDefinitionMappings:
                        resourceFields:
                        - name: host
                          jsonPath: .spec.host
                          secret: false
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        Then The resource registeredservices.primaza.io/$scenario_id-1:services is not available in cluster "worker"

    Scenario: Service Class removal will delete remote Registered Services
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id-1
                namespace: services
            spec:
                host: internal.db.stable.example.com
            """
        And On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id-2
                namespace: services
            spec:
                host: external.db.stable.example.com
            """
        And On Worker Cluster "worker", Resource is created
            """
            apiVersion: primaza.io/v1alpha1
            kind: ServiceClass
            metadata:
                name: $scenario_id-serviceclass
                namespace: services
            spec:
                constraints: {}
                resource:
                    apiVersion: stable.example.com/v1
                    kind: Backend
                    serviceEndpointDefinitionMappings:
                        resourceFields:
                        - name: host
                          jsonPath: .spec.host
                          secret: false
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        And  The resource registeredservices.primaza.io/$scenario_id-1:primaza-system is available in cluster "main"
        And  The resource registeredservices.primaza.io/$scenario_id-2:primaza-system is available in cluster "main"
        When The resource serviceclasses.primaza.io/$scenario_id-serviceclass:services is deleted from the cluster "worker"
        Then The resource registeredservices.primaza.io/$scenario_id-1:services is not available in cluster "worker"
        And  The resource registeredservices.primaza.io/$scenario_id-2:services is not available in cluster "worker"
        And  The resource registeredservices.primaza.io/$scenario_id-1:primaza-system is not available in cluster "main"
        And  The resource registeredservices.primaza.io/$scenario_id-2:primaza-system is not available in cluster "main"

    Scenario: A Service Class creates the secret associated with the registered service
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id
                namespace: services
            spec:
                host: internal.db.stable.example.com
            """
        When On Worker Cluster "worker", Resource is created
            """
            apiVersion: primaza.io/v1alpha1
            kind: ServiceClass
            metadata:
                name: $scenario_id-serviceclass
                namespace: services
            spec:
                constraints: {}
                resource:
                    apiVersion: stable.example.com/v1
                    kind: Backend
                    serviceEndpointDefinitionMappings:
                        resourceFields:
                        - name: host
                          jsonPath: .spec.host
                          secret: true
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        Then The resource registeredservices.primaza.io/$scenario_id:services is available in cluster "worker"
        And  The resource registeredservices.primaza.io/$scenario_id:primaza-system is available in cluster "main"
        And  jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id:primaza-system" in cluster main is "{"name": "host", "valueFromSecret": {"key": "host", "name": "$scenario_id-descriptor"}}"
        And  The resource secrets/$scenario_id-descriptor:primaza-system is available in cluster "main"
        And  jsonpath ".data.host" on "secrets/$scenario_id-descriptor:primaza-system" in cluster main is ""aW50ZXJuYWwuZGIuc3RhYmxlLmV4YW1wbGUuY29t""

    Scenario: A deleted Service Class with secret generation also removes the secret
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id
                namespace: services
            spec:
                host: internal.db.stable.example.com
            """
        And On Worker Cluster "worker", Resource is created
            """
            apiVersion: primaza.io/v1alpha1
            kind: ServiceClass
            metadata:
                name: $scenario_id-serviceclass
                namespace: services
            spec:
                constraints: {}
                resource:
                    apiVersion: stable.example.com/v1
                    kind: Backend
                    serviceEndpointDefinitionMappings:
                        resourceFields:
                        - name: host
                          jsonPath: .spec.host
                          secret: true
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        And  The resource registeredservices.primaza.io/$scenario_id:primaza-system is available in cluster "main"
        And  The resource secrets/$scenario_id-descriptor:primaza-system is available in cluster "main"
        When The resource serviceclasses.primaza.io/$scenario_id-serviceclass:services is deleted from the cluster "worker"
        Then The resource registeredservices.primaza.io/$scenario_id:services is not available in cluster "worker"
        And  The resource secrets/$scenario_id-descriptor:services is not available in cluster "worker"
        And  The resource registeredservices.primaza.io/$scenario_id:primaza-system is not available in cluster "main"
        And  The resource secrets/$scenario_id-descriptor:primaza-system is not available in cluster "main"
