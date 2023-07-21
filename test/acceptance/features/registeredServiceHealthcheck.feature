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
        Then On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "<state>"

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
        And On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Unreachable"
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
        Then On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Available"
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
        And On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Available"
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
        Then On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Unknown"
        # have polling deal with the wait here
        And On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Available"

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
        And On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Unreachable"
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
        Then On Primaza Cluster "main", RegisteredService "$scenario_id" state will eventually move to "Available"
