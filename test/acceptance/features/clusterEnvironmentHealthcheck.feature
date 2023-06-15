Feature: ClusterEnvironment's Healthchecks

    Background:
        Given Primaza Cluster "main" is running
        And On Primaza Cluster "main", "configmap" named "primaza-manager-config" in "primaza-system" is patched
        """
        {
            "data": {
                "health-check-interval": "10"
            }
        }
        """
        And On Primaza Cluster "main", controller manager is deleted
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
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
        """
        And On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Online"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Status "True"

    Scenario: Status change for ClusterEnvironment: ClusterContextSecret is deleted
        When On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: primaza-kw
            namespace: primaza-system
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Reason "ErrorDuringHealthCheck"

    Scenario: Status change for ClusterEnvironment: ClusterContextSecret is updated incorrectly
        When On Primaza Cluster "main", "secret" named "primaza-kw" in "primaza-system" is patched
        """
        {
            "data": {
                "kubeconfig": ""
            }
        }
        """
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Reason "ErrorDuringHealthCheck"

    Scenario: Status change for ClusterEnvironment: Worker cluster is deleted
        When Worker Cluster "worker" is deleted
        Then On Primaza Cluster "main", ClusterEnvironment "worker" state will eventually move to "Offline"
        And On Primaza Cluster "main", ClusterEnvironment "worker" status condition with Type "Online" has Reason "HealthCheckFailed"

