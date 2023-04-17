Feature: Service Agent watches Service Class Resource

    Background:
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Worker Cluster "worker", service namespace "services" exists
        And On Worker Cluster "worker", Primaza Service Agent is deployed into namespace "services"
        And Primaza cluster's "main" kubeconfig is available on "worker" in namespace "services"
        And Resource "backend_crd.yaml" is installed on worker cluster "worker" in namespace "services"
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
                    secret: false
            serviceClassIdentity:
              - name: type
                value: backend
              - name: provider
                value: stable.example.com
              - name: version
                value: v1
        """

    Scenario: A Service Class creates Registered Services as specified
        When On Worker Cluster "worker", Resource is created
        """
        apiVersion: stable.example.com/v1
        kind: Backend
        metadata:
            name: $scenario_id-1
            namespace: services
        spec:
            host: internal.db.stable.example.com
        """
        Then On Primaza Cluster "main", RegisteredService "$scenario_id-1" is available
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-1:primaza-system" in cluster main is "{"name":"host","value":"internal.db.stable.example.com"}"

    Scenario: A Registered Service should not be created if the resource does not  have the needed binding information
        When On Worker Cluster "worker", Resource is created
        """
        apiVersion: stable.example.com/v1
        kind: Backend
        metadata:
            name: $scenario_id-1
            namespace: services
        spec:
            host_internal_db: internal.db.stable.example.com
        """
        Then On Primaza Cluster "main", RegisteredService "scenario_id-1" is not available

    Scenario: A Service Class updates Registered Services
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
        And On Primaza Cluster "main", RegisteredService "$scenario_id-1" is available
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-1:primaza-system" in cluster main is "{"name":"host","value":"internal.db.stable.example.com"}"
        When On Worker Cluster "worker", Resource is updated
        """
        apiVersion: stable.example.com/v1
        kind: Backend
        metadata:
            name: $scenario_id-1
            namespace: services
        spec:
            host: internal-upd.db.stable.example.com
        """
        Then On Primaza Cluster "main", RegisteredService "$scenario_id-1" is available
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-1:primaza-system" in cluster main is "{"name":"host","value":"internal-upd.db.stable.example.com"}"

    Scenario: A Service Class deletes Registered Services
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
        And On Primaza Cluster "main", RegisteredService "$scenario_id-1" is available
        And jsonpath ".spec.serviceEndpointDefinition[0]" on "registeredservices.primaza.io/$scenario_id-1:primaza-system" in cluster main is "{"name":"host","value":"internal.db.stable.example.com"}"
        When On Worker Cluster "worker", Resource is deleted
        """
        apiVersion: stable.example.com/v1
        kind: Backend
        metadata:
            name: $scenario_id-1
            namespace: services
        """
        Then On Primaza Cluster "main", RegisteredService "scenario_id-1" is not available
