Feature: Service Binding Environment Projection

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
        And On Worker Cluster "worker", test application "applicationone" with label "myapp" is running in namespace "applications"
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
          envs:
          - name: MY_PASSWORD
            key: password
          - name: MY_HOST
            key: host
          - name: MY_PORT
            key: port
          target:
            environmentTag: stage
          application:
            apiVersion: apps/v1
            kind: Deployment
            selector:
              matchLabels:
                app: myapp
        """
        Then On Primaza Cluster "main", the status of ServiceClaim "sc-test" is "Resolved"

    Scenario: Create a service binding with environment projection

        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And  On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        And On Worker Cluster "worker", Service Binding "sc-test" exists in "applications"
        And On Worker Cluster "worker", ServiceBinding "sc-test" in namespace "applications" state will eventually move to "Ready"

    Scenario: Service binding resource being deleted

        And  On Primaza Cluster "main", RegisteredService "primaza-rsdb" state will eventually move to "Claimed"
        And  On Primaza Cluster "main", ServiceCatalog "stage" will not contain RegisteredService "primaza-rsdb"
        And  On Worker Cluster "worker", the secret "sc-test" in namespace "applications" has the key "type" with value "psqlserver"
        And On Worker Cluster "worker", Service Binding "sc-test" exists in "applications"
        And On Worker Cluster "worker", ServiceBinding "sc-test" in namespace "applications" state will eventually move to "Ready"
        When On Worker Cluster "worker", Resource is deleted
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: sc-test
            namespace: applications
        """
        Then On Primaza Cluster "main", file "/bindings/newapp-binding/username" is unavailable in application pod with label "myapp" running in namespace "applications"
