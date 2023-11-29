Feature: Claim from an application namespace (Pull)

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
            synchronizationStrategy: Pull
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
        And On Primaza Cluster "main", Resource is created
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

    Scenario: Claim with label selector from an application namespace
        Given On Cluster "main", logs of deployment "primaza-controller-manager" in "primaza-system" contain
        """
        INFO\sinformer synced\s{"controller": "clusterenvironment
        """
        And On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ApplicationServiceClaim
        metadata:
          name: sc-test
          namespace: applications
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
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        Then On Primaza Cluster "main", ControlPlaneServiceClaim "sc-test" state will eventually move to "Resolved"
        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And  On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And  On Worker Cluster "worker", the status of ApplicationServiceClaim "sc-test" is "Resolved"
        And  On Worker Cluster "worker", the RegisteredService bound to the ApplicationServiceClaim "sc-test" is "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"

    Scenario: Delete claim with label selector from an application namespace
        Given On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ApplicationServiceClaim
        metadata:
          name: sc-test
          namespace: applications
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
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        And On Primaza Cluster "main", ControlPlaneServiceClaim "sc-test" state will eventually move to "Resolved"
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        When The resource applicationserviceclaims.primaza.io/sc-test:applications is deleted from the cluster "worker"
        Then The resource controlplaneserviceclaims.primaza.io/sc-test:primaza-system is not available in cluster "main"
