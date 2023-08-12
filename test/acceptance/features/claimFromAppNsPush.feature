Feature: Claim from an application namespace (Push)

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

    Scenario: Claim with label selector from an application namespace not resolved
        Given On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        Then On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Pending"
        And  On Worker Cluster "worker", the status of ServiceClaim "sc-test" is "Pending"

    Scenario: Claim with label selector from an application namespace
        Given On Primaza Cluster "main", Resource is created
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
        And On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        Then On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Resolved"
        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And  On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And  On Primaza Cluster "main", the RegisteredService bound to the ServiceClaim "sc-test" is "primaza-rsdb"
        And  On Worker Cluster "worker", the status of ServiceClaim "sc-test" is "Resolved"
        And  On Worker Cluster "worker", the RegisteredService bound to the ServiceClaim "sc-test" is "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"

    Scenario: Update claim with label selector from an application namespace
        Given On Primaza Cluster "main", Resource is created
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
        And On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        And On Worker Cluster "worker", the status of ServiceClaim "sc-test" is "Resolved"
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        And On Worker Cluster "worker", Resource is updated
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
        metadata:
          name: sc-test
          namespace: applications
        spec:
          serviceClassIdentity:
          - name: type
            value: mysql
          - name: provider
            value: aws
          serviceEndpointDefinitionKeys:
          - host
          - port
          - user
          - password
          - database
          environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        And On Worker Cluster "worker", the status of ServiceClaim "sc-test" is "Invalid"
        And jsonpath ".status.conditions[0].reason" on "serviceclaims.primaza.io/sc-test:applications" in cluster worker is "ValidationError"

    Scenario: Delete claim with label selector from an application namespace
        Given On Primaza Cluster "main", Resource is created
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
        And On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        And On Worker Cluster "worker", the status of ServiceClaim "sc-test" is "Resolved"
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        When The resource serviceclaims.primaza.io/sc-test:applications is deleted from the cluster "worker"
        Then The resource serviceclaims.primaza.io/sc-test:primaza-system is not available in cluster "main"

    Scenario: Service Catalog is updated with the services claimed by labels
        Given On Primaza Cluster "main", Resource is created
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
        And On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
        Then On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Resolved"
        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And  On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And  On Primaza Cluster "main", ServiceCatalog "stage" contain RegisteredService "primaza-rsdb" in claimedByLabels
        And  On Primaza Cluster "main", the RegisteredService bound to the ServiceClaim "sc-test" is "primaza-rsdb"
        And  On Worker Cluster "worker", the status of ServiceClaim "sc-test" is "Resolved"
        And  On Worker Cluster "worker", the RegisteredService bound to the ServiceClaim "sc-test" is "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
