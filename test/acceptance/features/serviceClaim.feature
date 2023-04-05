Feature: Service claim with label selector

    Scenario: Create a service claim with label selector
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
          """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
        And  On Primaza Cluster "main", ServiceCatalog "primaza-service-catalog" will not contain RegisteredService "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"

    Scenario: Create a service claim with label selector, no constraints
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
          """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
        And  On Primaza Cluster "main", ServiceCatalog "primaza-service-catalog" will not contain RegisteredService "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"

    Scenario: Create a service claim with label selector, no constraints, sci subset and sed subset
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
          """
        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
        metadata:
          name: sc-test
          namespace: primaza-system
        spec:
          serviceClassIdentity:
          - name: type
            value: psqlserver
          serviceEndpointDefinitionKeys:
          - host
          - port
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
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And On Primaza Cluster "main", ServiceCatalog "primaza-service-catalog" will not contain RegisteredService "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"

    Scenario: Create a service claim with non-existing SED key
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
          """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "main", ServiceCatalog "primaza-service-catalog" will contain RegisteredService "primaza-rsdb"

    Scenario: Create a service claim with non-matching SCI
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
        """
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
        kind: ServiceClaim
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
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "main", ServiceCatalog "primaza-service-catalog" will contain RegisteredService "primaza-rsdb"

    Scenario: Create ServiceClaim with ApplicationClusterContext and EnvironmentTag
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
          """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is not getting created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          environmentTag: stage
          applicationClusterContext:
            clusterEnvironmentName: worker
            namespace: applications
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """

    Scenario: Create ServiceClaim with empty ApplicationClusterContext and EnvironmentTag
        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And On Worker Cluster "worker", application namespace "applications" exists
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
              value: quedicelagente
            - name: database
              value: davpdata
          sla: L3
          """
        And On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        When On Primaza Cluster "main", Resource is not getting created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
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
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              matchLabels:
                a: b
                c: d
        """
