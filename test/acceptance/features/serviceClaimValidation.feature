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

    Scenario: Create ServiceClaim with ApplicationClusterContext and EnvironmentTag
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
          target:
            environmentTag: stage
            applicationClusterContext:
              clusterEnvironmentName: worker
              namespace: applications
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              byLabels:
                matchLabels:
                  a: b
                  c: d
        """

    Scenario: Create ServiceClaim with empty ApplicationClusterContext and EnvironmentTag
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
          target: {}
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              byLabels:
                matchLabels:
                  a: b
                  c: d
        """

    Scenario: Create ServiceClaim without target
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
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              byLabels:
                matchLabels:
                  a: b
                  c: d
        """
        Then jsonpath ".status.conditions[] | select(.type=="ValidResource") | .status" on "serviceclaims.primaza.io/sc-test:primaza-system" in cluster main is "False"

    Scenario: Create ServiceClaim with Application name and Application selector
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
          target:
            environmentTag: stage
          application:
            kind: Deployment
            apiVersion: apps/v1
            selector:
              byName: some-name
              byLabels:
                matchLabels:
                  a: b
                  c: d
        """
