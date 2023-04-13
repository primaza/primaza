Feature: Claim for an Application by Name

    Background
            Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for "main" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
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

    Scenario: Application exists with Claim for an Application by Name
        Given On Worker Cluster "worker", Resource is created
        """
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: stage-app
          namespace: applications
          labels:
            app: stage-app
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: stage-app
          template:
            metadata:
              labels:
                app: stage-app
            spec:
              containers:
              - name: bash
                image: bash:latest
                command: ["sleep","infinity"]
        """
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
            name: stage-app
        """
        Then On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Resolved"
        And On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        And On Worker Cluster "worker", ServiceBinding "sc-test" on namespace "applications" state will eventually move to "Ready"

    Scenario: Application does not exists with Claim for an Application by Name
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
            name: stage-app
        """
        Then On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Resolved"
        And On Worker Cluster "worker", ServiceBinding "sc-test" on namespace "applications" state will eventually move to "Malformed"
