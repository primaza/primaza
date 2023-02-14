Feature: Register a cloud service in Primaza cluster

    Scenario: Cloud Service Registration, no Healthcheck provided and no ServiceCatalog exists
        Given Primaza Cluster "primaza-main" is running
        When On Primaza Cluster "primaza-main", Resource is created
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
        Then On Primaza Cluster "primaza-main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "primaza-main", ServiceCatalog "primaza-service-catalog" will contain RegisteredService "primaza-rsdb"


Scenario: Cloud Service Registration, no Healthcheck provided and ServiceCatalog exists
        Given Primaza Cluster "primaza-main" is running
        And   On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceCatalog
        metadata:
          name: primaza-service-catalog
          namespace: primaza-system
        spec:
          services:
          - name: davprssql
            serviceClassIdentity:
            - name: type
              value: mysqlserver
            - name: provider
              value: aws
            serviceEndpointDefinitionKeys:
            - host
            - port
            - user
            - password
            - database
        """
        When On Primaza Cluster "primaza-main", Resource is created
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
        Then On Primaza Cluster "primaza-main", RegisteredService "primaza-rsdb" state will eventually move to "Available"
        And On Primaza Cluster "primaza-main", ServiceCatalog "primaza-service-catalog" will contain RegisteredService "primaza-rsdb"


Scenario: Cloud Service Registration, no Healthcheck provided, ServiceCatalog exists, and Registered Service deleted
        Given Primaza Cluster "primaza-main" is running
        And   On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceCatalog
        metadata:
          name: primaza-service-catalog
          namespace: primaza-system
        spec:
          services:
          - name: davprssql
            serviceClassIdentity:
            - name: type
              value: mysqlserver
            - name: provider
              value: aws
            serviceEndpointDefinitionKeys:
            - host
            - port
            - user
            - password
            - database
        """
        And   On Primaza Cluster "primaza-main", Resource is created
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
        When On Primaza Cluster "primaza-main", RegisteredService "primaza-rsdb" is deleted
        Then On Primaza Cluster "primaza-main", ServiceCatalog "primaza-service-catalog" will not contain RegisteredService "primaza-rsdb"


Scenario: Cloud Service Registration, no Healthcheck provided, ServiceCatalog does not exists, and Registered Service deleted
        Given Primaza Cluster "primaza-main" is running
        And   On Primaza Cluster "primaza-main", Resource is created
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
        When On Primaza Cluster "primaza-main", RegisteredService "primaza-rsdb" is deleted
        Then On Primaza Cluster "primaza-main", ServiceCatalog "primaza-service-catalog" will not contain RegisteredService "primaza-rsdb"
