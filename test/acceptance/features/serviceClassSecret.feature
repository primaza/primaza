Feature: ServicesClasses can extract RegisteredService from resource-linked secrets

    Background:
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
        And On Worker Cluster "worker", Primaza Service Agent is deployed into namespace "services"
        And Resource "backend_crd.yaml" is installed on worker cluster "worker" in namespace "services"
        And On Primaza Cluster "main", Resource is created
            """
            apiVersion: rbac.authorization.k8s.io/v1
            kind: RoleBinding
            metadata:
                name: primaza:reporter-svc-worker-services
                namespace: primaza-system
            roleRef:
                apiGroup: rbac.authorization.k8s.io
                kind: Role
                name: primaza-reporter
            subjects:
            - kind: ServiceAccount
              name: primaza-svc-worker-services
            """

    Scenario: A Service Class creates Registered Services as specified (JsonPath secretRefField)
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id
                namespace: services
            spec:
                fromSecret:
                - secretName: $scenario_id-sec
                  secretKey: internal-host
            ---
            apiVersion: v1
            kind: Secret
            metadata:
                name: $scenario_id-sec
                namespace: services
            stringData:
                internal-host: internal.db.stable.example.com
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
                        secretRefFields:
                        - name: host
                          secretName:
                            jsonPath: .spec.fromSecret[0].secretName
                          secretKey:
                            jsonPath: .spec.fromSecret[0].secretKey
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        Then The resource registeredservices.primaza.io/$scenario_id:primaza-system is available in cluster "main"
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id:primaza-system" in cluster main is "{"name": "host", "valueFromSecret": {"key": "host", "name": "$scenario_id-descriptor"}}"
        And The resource secrets/$scenario_id-descriptor:primaza-system is available in cluster "main"
        And jsonpath ".data.host" on "secrets/$scenario_id-descriptor:primaza-system" in cluster main is ""aW50ZXJuYWwuZGIuc3RhYmxlLmV4YW1wbGUuY29t""

    Scenario: A Service Class creates Registered Services as specified (Constant secretRefField)
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id
                namespace: services
            spec:
                fromSecret:
                - secretName: $scenario_id-sec
                  secretKey: internal-host
            ---
            apiVersion: v1
            kind: Secret
            metadata:
                name: $scenario_id-sec
                namespace: services
            stringData:
                internal-host: internal.db.stable.example.com
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
                        secretRefFields:
                        - name: host
                          secretName:
                            constant: $scenario_id-sec
                          secretKey:
                            constant: internal-host
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        Then The resource registeredservices.primaza.io/$scenario_id:primaza-system is available in cluster "main"
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id:primaza-system" in cluster main is "{"name": "host", "valueFromSecret": {"key": "host", "name": "$scenario_id-descriptor"}}"
        And The resource secrets/$scenario_id-descriptor:primaza-system is available in cluster "main"
        And jsonpath ".data.host" on "secrets/$scenario_id-descriptor:primaza-system" in cluster main is ""aW50ZXJuYWwuZGIuc3RhYmxlLmV4YW1wbGUuY29t""

    Scenario: A Registered Service isn't created when the secret doesn't exist
        Given On Worker Cluster "worker", Resource is created
            """
            apiVersion: stable.example.com/v1
            kind: Backend
            metadata:
                name: $scenario_id
                namespace: services
            spec:
                fromSecret:
                - secretName: $scenario_id-sec
                  secretKey: internal-host
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
                        secretRefFields:
                        - name: host
                          secretName:
                            jsonPath: .spec.fromSecret[0].secretName
                          secretKey:
                            jsonPath: .spec.fromSecret[0].secretKey
                serviceClassIdentity:
                  - name: type
                    value: backend
                  - name: provider
                    value: stable.example.com
                  - name: version
                    value: v1
            """
        Then The resource registeredservices.primaza.io/$scenario_id:primaza-system is not available in cluster "main"
