Feature: Healthcheck state transitions for Registered Services

    Background:
        Given Primaza Cluster "main" is running

    Scenario Outline: Registered service state reflects healthcheck status
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          healthcheck:
            container:
              image: "alpine:latest"
              command:
              - /bin/sh
              - -c
              - <command>
              minutes: 1
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        Then On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "<state>" in "120" seconds

      Examples: Correct states
        | command     | state       |
        | "sleep 600" | Unknown     |
        | "exit 0"    | Available   |
        | "exit 1"    | Unreachable |

    Scenario: Removing a healthcheck immediately makes the registered service available
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          healthcheck:
            container:
              image: "alpine:latest"
              command:
              - /bin/sh
              - -c
              - exit 1
              minutes: 1
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        And On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Unreachable" in "120" seconds
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        Then On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Available" in "120" seconds
        And The resource cronjobs.batch/$scenario_id:primaza-system is not available in cluster "main"

    Scenario: Adding a healthcheck runs it before marking the service as available
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        And On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Available" in "120" seconds
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          healthcheck:
            container:
              image: "alpine:latest"
              command:
              - /bin/sh
              - -c
              - sleep 1 && exit 0
              minutes: 1
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        Then On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Unknown" in "120" seconds
        # have polling deal with the wait here
        And On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Available" in "120" seconds

    Scenario: Changing a healthcheck runs the healthcheck before making it available
        Given On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          healthcheck:
            container:
              image: "alpine:latest"
              command:
              - /bin/sh
              - -c
              - exit 1
              minutes: 1
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        And On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Unreachable" in "120" seconds
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: RegisteredService
        metadata:
          name: $scenario_id
          namespace: primaza-system
        spec:
          healthcheck:
            container:
              image: "alpine:latest"
              command:
              - /bin/sh
              - -c
              - exit 0
              minutes: 1
          serviceClassIdentity:
            - name: type
              value: psqlserver
            - name: provider
              value: aws
          serviceEndpointDefinition:
            - name: port
              value: "5432"
        """
        Then On Cluster "main", RegisteredService "$scenario_id" in namespace "primaza-system" state will move to "Available" in "120" seconds
