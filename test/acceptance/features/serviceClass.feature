Feature: Use ServicesClass resources to manage RegisteredService resources

    Background:
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Worker Cluster "worker", service namespace "services" exists
        And On Worker Cluster "worker", Primaza Service Agent is deployed into namespace "services"
        And Primaza cluster's "main" kubeconfig is available on "worker" in namespace "services"
        And Resource "backend_crd.yaml" is installed on worker cluster "worker" in namespace "services"

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
        Then The resource registeredservices.primaza.io/$scenario_id-1:primaza-system is available in cluster "main"
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-1:primaza-system" in cluster main is "{"name":"host","value":"internal.db.stable.example.com"}"
        And The resource registeredservices.primaza.io/$scenario_id-2:primaza-system is available in cluster "main"
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-2:primaza-system" in cluster main is "{"name":"host","value":"external.db.stable.example.com"}"

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
        And The resource registeredservices.primaza.io/$scenario_id-1:primaza-system is available in cluster "main"
        And The resource registeredservices.primaza.io/$scenario_id-2:primaza-system is available in cluster "main"
        When The resource serviceclasses.primaza.io/$scenario_id-serviceclass:services is deleted from the cluster "worker"
        Then The resource registeredservices.primaza.io/$scenario_id-1:primaza-system is not available in cluster "main"
        Then The resource registeredservices.primaza.io/$scenario_id-2:primaza-system is not available in cluster "main"
