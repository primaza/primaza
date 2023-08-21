Feature: Service claim with label selector

    Background:
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
            environmentName: stage
            clusterContextSecret: primaza-kw
            applicationNamespaces:
            - applications
        """
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: $scenario_id
            namespace: primaza-system
        stringData:
            password: quedicelagente
        """

    Scenario: Service Claim does not reuse an already claimed service
        Given On Primaza Cluster "main", Resource is created
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
              valueFromSecret:
                name: $scenario_id
                key: password
            - name: database
              value: davpdata
          sla: L3
        """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ControlPlaneServiceClaim
        metadata:
          name: sc-test
          namespace: primaza-system
        spec:
          serviceClassIdentity:
          - name: type
            value: psqlserver
          - name: provider
            value: aws
          serviceEndpointDefinitionKeys:
          - host
          - port
          - user
          - password
          - database
          target:
            environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        And On Primaza Cluster "main", the status of ControlPlaneServiceClaim "sc-test" is "Resolved"
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ControlPlaneServiceClaim
        metadata:
          name: sc-test-2
          namespace: primaza-system
        spec:
          serviceClassIdentity:
          - name: type
            value: psqlserver
          - name: provider
            value: aws
          serviceEndpointDefinitionKeys:
          - host
          - port
          - user
          - password
          - database
          target:
            environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        And On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        And On Primaza Cluster "main", the status of ControlPlaneServiceClaim "sc-test-2" is "Pending"

    Scenario: Create a service claim with non-existing SED key
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: primaza-rsdb
          namespace: primaza-system
        spec:
          constraints:
            environments:
            - stage
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
              valueFromSecret:
                name: $scenario_id
                key: password
            - name: database
              value: davpdata
          sla: L3
        """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ControlPlaneServiceClaim
        metadata:
          name: sc-test
          namespace: primaza-system
        spec:
          serviceClassIdentity:
          - name: type
            value: psqlserver
          - name: provider
            value: aws
          serviceEndpointDefinitionKeys:
          - host
          - port
          - username
          - password
          - database
          target:
            environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        Then On Primaza Cluster "main", the status of ControlPlaneServiceClaim "sc-test" is "Pending"
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "main", ServiceCatalog "stage" will contain RegisteredService "primaza-rsdb"
        And jsonpath ".status.conditions[] | select(.type=="Ready") | .reason" on "controlplaneserviceclaims.primaza.io/sc-test:primaza-system" in cluster main is "NoMatchingServiceFound"

    Scenario: Create a service claim with non-matching SCI
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: primaza-rsdb
          namespace: primaza-system
        spec:
          constraints:
            environments:
            - stage
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
              valueFromSecret:
                name: $scenario_id
                key: password
            - name: database
              value: davpdata
          sla: L3
        """
        And On Primaza Cluster "main", Primaza ClusterContext secret "primaza-km" is published
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: main
            namespace: primaza-system
        spec:
            environmentName: stage
            clusterContextSecret: primaza-km
        """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ControlPlaneServiceClaim
        metadata:
          name: sc-test
          namespace: primaza-system
        spec:
          serviceClassIdentity:
          - name: type
            value: psqlserver
          - name: provider
            value: azure
          serviceEndpointDefinitionKeys:
          - host
          - port
          - user
          - password
          - database
          target:
            environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        Then On Primaza Cluster "main", the status of ControlPlaneServiceClaim "sc-test" is "Pending"
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "main", ServiceCatalog "stage" will contain RegisteredService "primaza-rsdb"
        And jsonpath ".status.conditions[] | select(.type=="Ready") | .reason" on "controlplaneserviceclaims.primaza.io/sc-test:primaza-system" in cluster main is "NoMatchingServiceFound"
