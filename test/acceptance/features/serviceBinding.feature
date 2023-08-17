Feature: Bind application to the secret pushed by agent app controller

    Scenario: Service binding projection of a secret into an application with direct reference

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "newapp" with label "myapp" is running in namespace "applications"
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
        Then On Primaza Cluster "main", ServiceBinding "newapp-binding" in namespace "applications" state will eventually move to "Ready"
        And  On Primaza Cluster "main", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/newapp-binding/username" has content "AzureDiamond"

    Scenario: Agents bind application on Worker Cluster

        Given Primaza Cluster "main" is running
        And   Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And   Clusters "main" and "worker" can communicate
        And   On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And   On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Resource is created
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
        And   On Worker Cluster "worker", test application "app" with label "myapp" is running in namespace "applications"
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
        Then On Worker Cluster "worker", ServiceBinding "app-binding" in namespace "applications" state will eventually move to "Ready"
        And  On Worker Cluster "worker", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/app-binding/username" has content "AzureDiamond"
        And  On Worker Cluster "worker", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/app-binding/password" has content "pass"

    Scenario: Service binding projection works for multiple applications matching the labels

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "applicationone" with label "myapp" is running in namespace "applications"
        And   On Primaza Cluster "main", test application "applicationtwo" with label "myapp" is running in namespace "applications"
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
        Then On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" state will eventually move to "Ready"
        And  On Primaza Cluster "main", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"

    Scenario: Service binding resource being deleted

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "newapp" with label "myapp" is running in namespace "applications"
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
        And  On Primaza Cluster "main", ServiceBinding "newapp-binding" in namespace "applications" state will eventually move to "Ready"
        When On Primaza Cluster "main", Resource is deleted
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
        Then On Primaza Cluster "main", file "/bindings/newapp-binding/username" is unavailable in application pod with label "myapp" running in namespace "applications"

    Scenario: Application Agent watches ServiceBindings' resources and projection works for application created after ServiceBinding creation

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And On Primaza Cluster "main", Resource is created
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
        When On Primaza Cluster "main", test application "applicationone" with label "myapp" is running in namespace "applications"
        Then On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" state will eventually move to "Ready"
        And On Primaza Cluster "main", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"

    Scenario: Service binding status updated when application is deleted

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And On Primaza Cluster "main", Resource is created
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
        And On Primaza Cluster "main", test application "applicationone" with label "myapp" is running in namespace "applications"
        And On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" state will eventually move to "Ready"
        And On Primaza Cluster "main", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"
        When The resource deployments/applicationone:applications is deleted from the cluster "main"
        Then jsonpath ".status.conditions[] | select(.type=="Bound") | .reason" on "servicebindings.primaza.io/application-binding:applications" in cluster main is "NoMatchingWorkloads"
        And On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" state will eventually move to "Ready"

    Scenario: ServiceBinding by label is not mounted for non matching label value

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "applicationone" with label "myapp" is running in namespace "applications"
        And   On Primaza Cluster "main", test application "applicationtwo" with label "myapp" is running in namespace "applications"
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
        And  On Primaza Cluster "main", test application "applicationthree" with label "myapplication" is running in namespace "applications"
        Then On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" state will eventually move to "Ready"
        And  On Primaza Cluster "main", in demo application's pod with label "myapp" running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"
        And On Primaza Cluster "main", file "/bindings/application-binding/username" is unavailable in application pod with label "myapplication" running in namespace "applications"

    Scenario: ServiceBinding by application with multiple label

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And   On Primaza Cluster "main", Resource is created
        """
        apiVersion: apps/v1
        kind: Deployment
        metadata:
            name: applicationone
            namespace: applications
            labels:
                app: applicationone
                mylabel: myapp
        spec:
            replicas: 1
            selector:
                matchLabels:
                    app: applicationone
            template:
                metadata:
                    labels:
                        app: applicationone
                spec:
                    containers:
                    - name: myapp
                      image: quay.io/service-binding/generic-test-app:20220216
        """
        And   On Primaza Cluster "main", Resource is created
        """
        apiVersion: apps/v1
        kind: Deployment
        metadata:
            name: applicationtwo
            namespace: applications
            labels:
                app: applicationtwo
                mylabel: myapp
        spec:
            replicas: 1
            selector:
                matchLabels:
                    app: applicationtwo
            template:
                metadata:
                    labels:
                        app: applicationtwo
                spec:
                    containers:
                    - name: myapp
                      image: quay.io/service-binding/generic-test-app:20220216
        """
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
                        mylabel: myapp
        """
        Then On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" state will eventually move to "Ready"
        And  On Primaza Cluster "main", in demo application's pod with label "applicationone" running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"
        And  On Primaza Cluster "main", in demo application's pod with label "applicationtwo" running in namespace "applications" file "/bindings/application-binding/username" has content "AzureDiamond"
        And  On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" is bound to workload "applicationone"
        And  On Primaza Cluster "main", ServiceBinding "application-binding" in namespace "applications" is bound to workload "applicationtwo"

    Scenario: Service Binding without application name and labels selector

        Given Primaza Cluster "main" is running
        And   On Primaza Cluster "main", application namespace "applications" for ClusterEnvironment "worker" exists
        And   On Primaza Cluster "main", Primaza Application Agent is deployed into namespace "applications"
        And   On Primaza Cluster "main", test application "newapp" with label "myapp" is running in namespace "applications"
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
        When On Primaza Cluster "main", Resource is not getting created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ServiceBinding
        metadata:
            name: newapp-binding
            namespace: applications
        spec:
            serviceEndpointDefinitionSecret: demo
            application:
                apiVersion: apps/v1
                kind: Deployment
        """
