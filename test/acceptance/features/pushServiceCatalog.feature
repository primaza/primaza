Feature: Push ServiceCatalog

  Background:
      Given Primaza Cluster "main" is running
      And Worker Cluster "worker" for "main" is running
      And Clusters "main" and "worker" can communicate
      And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" is published
      And On Worker Cluster "worker", application namespace "applications" exists

  Scenario: Initialize ServiceCatalog
      And On Primaza Cluster "main", Resource is created
      """
      apiVersion: primaza.io/v1alpha1
      kind: ClusterEnvironment
      metadata:
          name: worker
          namespace: primaza-system
      spec:
          environmentName: dev
          clusterContextSecret: primaza-kw
          applicationNamespaces:
          - applications
      """
      And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
      Then On Worker Cluster "worker", ServiceCatalog "dev" exists in "applications"
      And  On Primaza Cluster "main", ServiceCatalog "dev" exists

  Scenario: Add Registered Service to Service Catalog
      When On Primaza Cluster "main", Resource is created
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
      And On Primaza Cluster "main", Resource is created
      """
      apiVersion: primaza.io/v1alpha1
      kind: ClusterEnvironment
      metadata:
          name: worker
          namespace: primaza-system
      spec:
          environmentName: dev
          clusterContextSecret: primaza-kw
          applicationNamespaces:
          - applications
      """
      And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
      And  On Primaza Cluster "main", ServiceCatalog "dev" exists
      And  On Worker Cluster "worker", ServiceCatalog "dev" exists in "applications"
      Then On Primaza Cluster "main", ServiceCatalog "dev" will contain RegisteredService "primaza-rsdb"
      And  On Worker Cluster "worker", ServiceCatalog "dev" in application namespace "applications" will contain RegisteredService "primaza-rsdb"

  Scenario: Update Registered Service in Service Catalog
      And On Primaza Cluster "main", Resource is created
      """
      apiVersion: primaza.io/v1alpha1
      kind: ClusterEnvironment
      metadata:
          name: worker
          namespace: primaza-system
      spec:
          environmentName: dev
          clusterContextSecret: primaza-kw
          applicationNamespaces:
          - applications
      """
      And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ApplicationNamespacePermissionsRequired" has Status "False"
      And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "ServiceNamespacePermissionsRequired" has Status "False"
      And On Worker Cluster "worker", ServiceCatalog "dev" exists in "applications"
      And On Primaza Cluster "main", ServiceCatalog "dev" exists
      And On Primaza Cluster "main", Resource is updated
      """
      apiVersion: primaza.io/v1alpha1
      kind: ServiceCatalog
      metadata:
        name: dev
        namespace: primaza-system
      spec:
        services:
        - name: primaza-rsdb
          serviceClassIdentity:
          - name: type
            value: mysqlserver
          serviceEndpointDefinitionKeys:
          - host
          - port
          - user
          - password
          - database
      """
      And On Primaza Cluster "main", ServiceCatalog "dev" will contain RegisteredService "primaza-rsdb"
      And On Worker Cluster "worker", ServiceCatalog "dev" in application namespace "applications" will contain RegisteredService "primaza-rsdb"
      When On Primaza Cluster "main", Resource is updated
      """
      apiVersion: primaza.io/v1alpha1
      kind: ServiceCatalog
      metadata:
        name: dev
        namespace: primaza-system
      spec:
        services:
        - name: primaza-rsdb
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
      Then On Primaza Cluster "main", ServiceCatalog "dev" has RegisteredService "primaza-rsdb" with Service Class Identity "provider=aws"
      And On Worker Cluster "worker", ServiceCatalog "dev" in application namespace "applications" has "primaza-rsdb" with "provider=aws"

  Scenario: Remove Registered Service from Service Catalog
      Given On Primaza Cluster "main", Resource is created
      """
      apiVersion: primaza.io/v1alpha1
      kind: ClusterEnvironment
      metadata:
          name: worker
          namespace: primaza-system
      spec:
          environmentName: dev
          clusterContextSecret: primaza-kw
          applicationNamespaces:
          - applications
      """
      And On Primaza Cluster "main", Resource is updated
      """
      apiVersion: primaza.io/v1alpha1
      kind: ServiceCatalog
      metadata:
        name: dev
        namespace: primaza-system
      spec:
        services:
        - name: primaza-rsdb
          serviceClassIdentity:
          - name: type
            value: mysqlserver
          serviceEndpointDefinitionKeys:
          - host
          - port
          - user
          - password
          - database
      """
      And  On Primaza Cluster "main", ServiceCatalog "dev" will contain RegisteredService "primaza-rsdb"
      When On Primaza Cluster "main", Resource is updated
      """
      apiVersion: primaza.io/v1alpha1
      kind: ServiceCatalog
      metadata:
        name: dev
        namespace: primaza-system
      """
      Then On Primaza Cluster "main", ServiceCatalog "dev" will not contain RegisteredService "primaza-rsdb"
      And  On Worker Cluster "worker", ServiceCatalog "dev" in application namespace "applications" will not contain RegisteredService "primaza-rsdb"
