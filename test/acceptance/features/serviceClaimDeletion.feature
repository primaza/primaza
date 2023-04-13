Feature: On claim deletion, remove Bindings

    Scenario: Delete an active claim
        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for "main" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
        And   On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Resource is created
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
        And On Primaza Cluster "main", Resource is created
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
        And  On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Resolved"
        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And  On Worker Cluster "worker", Service Binding "sc-test" exists in "applications"
        When On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceClaim
        metadata:
          name: sc-test
          namespace: primaza-system
        """
        Then On Worker Cluster "worker", Service Binding "sc-test" does not exist in "applications"
        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
