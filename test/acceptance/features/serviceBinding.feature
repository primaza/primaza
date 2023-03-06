Feature: Bind application to the secret pushed by agent app controller

    Scenario: Service binding projection of a secret into an application with direct reference

        Given   Primaza Cluster "primaza-main" is running
        And     On Primaza Cluster "primaza-main", namespace "applications" exists
        And     On Primaza Cluster "primaza-main", Primaza Application Agent is deployed into namespace "applications"
        Given   On Primaza Cluster "primaza-main", test application "newapp" is running in namespace "applications"
        And     On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
        """
        When    On Primaza Cluster "primaza-main", Resource is created
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
        Then  On Primaza Cluster "primaza-main", ServiceBinding "newapp-binding" on namespace "applications" state will eventually move to "Ready"
        And   On Primaza Cluster "primaza-main", in demo application's pod running in namespace "applications" file "/bindings/newapp-binding/username" has content "AzureDiamond"

    Scenario: Agents bind application on Worker Cluster

        Given Worker Cluster "primaza-worker" is running
        And   On Worker Cluster "primaza-worker", application namespace "applications" exists
        And   On Worker Cluster "primaza-worker", Primaza Application Agent is deployed into namespace "applications"
        And   On Worker Cluster "primaza-worker", test application "app" is running in namespace "applications"
        And   On Worker Cluster "primaza-worker", Resource is created
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
        When On Worker Cluster "primaza-worker", Resource is created
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
        Then On Worker Cluster "primaza-worker", ServiceBinding "app-binding" on namespace "applications" state will eventually move to "Ready"
        And   On Worker Cluster "primaza-worker", in demo application's pod running in namespace "applications" file "/bindings/app-binding/username" has content "AzureDiamond"
        And   On Worker Cluster "primaza-worker", in demo application's pod running in namespace "applications" file "/bindings/app-binding/password" has content "pass"

    Scenario: Service binding projection works for multiple applications matching the labels

        Given   Primaza Cluster "primaza-main" is running
        And     On Primaza Cluster "primaza-main", namespace "applications" exists
        And     On Primaza Cluster "primaza-main", Primaza Application Agent is deployed into namespace "applications"
        Given   On Primaza Cluster "primaza-main", test application "applicationone" is running in namespace "applications"
        And     On Primaza Cluster "primaza-main", test application "applicationtwo" is running in namespace "applications"
        And     On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
        """
        When    On Primaza Cluster "primaza-main", Resource is created
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
        Then  On Primaza Cluster "primaza-main", ServiceBinding "application-binding" on namespace "applications" state will eventually move to "Ready"
        And   On Primaza Cluster "primaza-main", in demo application's pod running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"

    Scenario: Service binding resource being deleted

        Given   Primaza Cluster "primaza-main" is running
        And     On Primaza Cluster "primaza-main", namespace "applications" exists
        And     On Primaza Cluster "primaza-main", Primaza Application Agent is deployed into namespace "applications"
        Given   On Primaza Cluster "primaza-main", test application "newapp" is running in namespace "applications"
        And     On Primaza Cluster "primaza-main", Resource is created
        """
        apiVersion: v1
        kind: Secret
        metadata:
            name: demo
            namespace: applications
        stringData:
            username: AzureDiamond
        """
        And    On Primaza Cluster "primaza-main", Resource is created
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
        And  On Primaza Cluster "primaza-main", ServiceBinding "newapp-binding" on namespace "applications" state will eventually move to "Ready"
        Then On Primaza Cluster "primaza-main", Resource is deleted
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
        And On Primaza Cluster "primaza-main", file "/bindings/newapp-binding/username" is unavailable in application pod running in namespace "applications"
