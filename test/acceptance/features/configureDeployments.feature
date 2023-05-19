Feature: Configure Incorrect data for Service and Application agent deployments

    Scenario: Modifying Manager Config Map, Primaza's Agent are not deployed successfully

        Given Primaza Cluster "main" is running
        And Worker Cluster "worker" for ClusterEnvironment "worker" is running
        And Clusters "main" and "worker" can communicate
        And On Primaza Cluster "main", Worker "worker"'s ClusterContext secret "primaza-kw" for ClusterEnvironment "worker" is published
        And On Primaza Cluster "main", Resource is updated
        """
        apiVersion: v1
        kind: ConfigMap
        metadata:
            name: primaza-manager-config
            namespace: primaza-system
        data:
            agentapp-manifest: |-
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                    name: primaza-app-agent
                    labels:
                        app.kubernetes.io/part-of: primaza
                        control-plane: primaza-app-agent
                    finalizers:
                    - agentapp.primaza.io/finalizer
                spec:
                    selector:
                        matchLabels:
                            control-plane: primaza-app-agent
                    replicas: 1
                    template:
                        metadata:
                            annotations:
                                kubectl.kubernetes.io/default-container: manager
                            labels:
                                control-plane: primaza-app-agent
                        spec:
                            securityContext:
                                runAsNonRoot: true
                            containers:
                                - command:
                                    - /manager
                                    args:
                                    - --leader-elect
                                    image: agentapp:notexist
                                    imagePullPolicy: IfNotPresent
                                    name: manager
                                    env:
                                        - name: WATCH_NAMESPACE
                                            valueFrom:
                                                fieldRef:
                                                    fieldPath: metadata.namespace
                                    securityContext:
                                        allowPrivilegeEscalation: false
                                            capabilities:
                                                drop:
                                                - "ALL"
                                    livenessProbe:
                                        httpGet:
                                            path: /healthz
                                            port: 8081
                                        initialDelaySeconds: 15
                                        periodSeconds: 20
                                    readinessProbe:
                                        httpGet:
                                            path: /readyz
                                            port: 8081
                                        initialDelaySeconds: 5
                                        periodSeconds: 10
                                    resources:
                                        limits:
                                            cpu: 500m
                                            memory: 128Mi
                                        requests:
                                            cpu: 10m
                                            memory: 64Mi
                            serviceAccountName: primaza-app-agent
                            terminationGracePeriodSeconds: 10
            agentsvc-manifest: |-
                apiVersion: apps/v1
                kind: Deployment
                metadata:
                name: primaza-svc-agent
                labels:
                    app.kubernetes.io/part-of: primaza
                    control-plane: primaza-svc-agent
                finalizers:
                    - agent.primaza.io/finalizer
                spec:
                    replicas: 1
                    selector:
                        matchLabels:
                            control-plane: primaza-svc-agent
                    template:
                        metadata:
                            annotations:
                                kubectl.kubernetes.io/default-container: manager
                            labels:
                                control-plane: primaza-svc-agent
                        spec:
                            containers:
                                - command:
                                    - /manager
                                    args:
                                        - --leader-elect
                                    env:
                                    - name: WATCH_NAMESPACE
                                        valueFrom:
                                            fieldRef:
                                                fieldPath: metadata.namespace
                                    image: agentsvc:notexist
                                    imagePullPolicy: IfNotPresent
                                    livenessProbe:
                                        httpGet:
                                            path: /healthz
                                            port: 8081
                                        initialDelaySeconds: 15
                                        periodSeconds: 20
                                    name: manager
                                    readinessProbe:
                                        httpGet:
                                            path: /readyz
                                            port: 8081
                                        initialDelaySeconds: 5
                                        periodSeconds: 10
                                    resources:
                                        limits:
                                            cpu: 500m
                                            memory: 128Mi
                                        requests:
                                            cpu: 10m
                                            memory: 64Mi
                                    securityContext:
                                        allowPrivilegeEscalation: false
                                        capabilities:
                                            drop:
                                            - ALL
                                    volumeMounts:
                                    - mountPath: /tmp/k8s-webhook-server/serving-certs
                                        name: cert
                                        readOnly: true
                            securityContext:
                                runAsNonRoot: true
                            serviceAccountName: primaza-svc-agent
                            terminationGracePeriodSeconds: 10
                            volumes:
                            - name: cert
                            secret:
                                defaultMode: 420
                                secretName: webhook-server-cert
        """
        And On Primaza Cluster "main", controller manager is deleted
        And On Worker Cluster "worker", service namespace "services" for ClusterEnvironment "worker" exists
        And On Worker Cluster "worker", application namespace "applications" for ClusterEnvironment "worker" exists
        When On Primaza Cluster "main", Resource is created
        """
        apiVersion: primaza.io/v1alpha1
        kind: ClusterEnvironment
        metadata:
            name: worker
            namespace: primaza-system
        spec:
            environmentName: dev
            clusterContextSecret: primaza-kw
            serviceNamespaces:
            - "services"
            applicationNamespaces:
            - "applications"
        """
        Then On Worker Cluster "worker", Primaza Application Agent does not exist into namespace "applications"
        And On Worker Cluster "worker", Primaza Service Agent does not exist into namespace "services"