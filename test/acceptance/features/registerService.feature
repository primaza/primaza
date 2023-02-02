Feature: Register a cloud service in Primaza cluster

    Scenario: Cloud Service Registration, no Healthcheck provided
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
        
