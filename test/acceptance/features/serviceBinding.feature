Feature: Bind application to the secret pushed by agent app controller

    Scenario: Service binding projection of a secret into an application with direct reference

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", Primaza Application Agent for ClusterEnvironment "worker" is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "newapp" is running in namespace "applications"
        And   On Primaza Cluster "main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
        """
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: newapp-binding
            namespace: applications
        spec:
            serviceEndpointDefinitionSecret: demo
            application:
                name: newapp
                apiVersion: apps/v1
                kind: Deployment
        """
        Then On Primaza Cluster "main", ServiceBinding "newapp-binding" on namespace "applications" state will eventually move to "Ready"
        And  On Primaza Cluster "main", in demo application's pod running in namespace "applications" file "/bindings/newapp-binding/username" has content "AzureDiamond"

    Scenario: Agents bind application on Worker Cluster

        Given Worker Cluster "worker" is running
        And   On Worker Cluster "worker", Primaza Application Agent for ClusterEnvironment "worker" is deployed into namespace "applications"
        And   On Worker Cluster "worker", test application "app" is running in namespace "applications"
        And   On Worker Cluster "worker", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
            password: pass
        """
        When On Worker Cluster "worker", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: app-binding
            namespace: applications
        spec:
            serviceEndpointDefinitionSecret: demo
            application:
                name: app
                apiVersion: apps/v1
                kind: Deployment
        """
        Then On Worker Cluster "worker", ServiceBinding "app-binding" on namespace "applications" state will eventually move to "Ready"
        And   On Worker Cluster "worker", in demo application's pod running in namespace "applications" file "/bindings/app-binding/username" has content "AzureDiamond"
        And   On Worker Cluster "worker", in demo application's pod running in namespace "applications" file "/bindings/app-binding/password" has content "pass"

    Scenario: Service binding projection works for multiple applications matching the labels

        Given Primaza Cluster "main" is running
        And   On Worker Cluster "worker", Primaza Application Agent for ClusterEnvironment "worker" is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "applicationone" is running in namespace "applications"
        And   On Primaza Cluster "main", test application "applicationtwo" is running in namespace "applications"
        And   On Primaza Cluster "main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
        """
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: application-binding
            namespace: applications
        spec:
            serviceEndpointDefinitionSecret: demo
            application:
                apiVersion: apps/v1
                kind: Deployment
                selector:
                    matchLabels:
                        app: myapp
        """
        Then On Primaza Cluster "main", ServiceBinding "application-binding" on namespace "applications" state will eventually move to "Ready"
        And  On Primaza Cluster "main", in demo application's pod running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"

    Scenario: Service binding resource being deleted
        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", Primaza Application Agent for ClusterEnvironment "worker" is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "newapp" is running in namespace "applications"
        And   On Primaza Cluster "main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
        """
        And On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: newapp-binding
            namespace: applications
        spec:
            serviceEndpointDefinitionSecret: demo
            application:
                name: newapp
                apiVersion: apps/v1
                kind: Deployment
        """
        And  On Primaza Cluster "main", ServiceBinding "newapp-binding" on namespace "applications" state will eventually move to "Ready"
        Then On Primaza Cluster "main", Resource is deleted
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: newapp-binding
            namespace: applications
        spec:
            serviceEndpointDefinitionSecret: demo
            application:
                name: newapp
                apiVersion: apps/v1
                kind: Deployment
        """
        And On Primaza Cluster "main", file "/bindings/newapp-binding/username" is unavailable in application pod running in namespace "applications"